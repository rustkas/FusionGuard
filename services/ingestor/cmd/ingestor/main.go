package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fusionguard/pkg/telemetry"
	"github.com/fusionguard/services/ingestor/internal/config"
	"github.com/fusionguard/services/ingestor/internal/health"
	"github.com/fusionguard/services/ingestor/internal/storage"
)

type ingestService struct {
	cfg            *config.Config
	nc             *nats.Conn
	allowed        map[string]struct{}
	lastTimestamps map[string]int64
	storage        *storage.Storage
	wg             sync.WaitGroup
}

func newIngestService(cfg *config.Config) (*ingestService, error) {
	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(cfg.Sampling.Allowed))
	for _, ch := range cfg.Sampling.Allowed {
		allowed[ch] = struct{}{}
	}

	stor, err := storage.New(storage.Config{
		PostgresDSN: cfg.Storage.PostgresDSN,
		WriteRaw:    cfg.Storage.WriteRaw,
	})
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &ingestService{
		cfg:            cfg,
		nc:             nc,
		allowed:        allowed,
		lastTimestamps: map[string]int64{},
		storage:        stor,
	}, nil
}

func (s *ingestService) ingestHandler(w http.ResponseWriter, r *http.Request) {
	var req telemetry.TelemetryPoint
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := req.Valid(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, ch := range req.Channels {
		if _, ok := s.allowed[ch.Name]; !ok {
			http.Error(w, "channel not allowed", http.StatusBadRequest)
			return
		}
	}

	last := s.lastTimestamps[req.ShotID]
	if last != 0 && req.TsUnixNs <= last {
		http.Error(w, "timestamp must increase", http.StatusBadRequest)
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "serialization error", http.StatusInternalServerError)
		return
	}

	if err := s.nc.Publish(s.cfg.NATS.SubjectRaw, payload); err != nil {
		http.Error(w, "publish failure", http.StatusInternalServerError)
		return
	}

	s.lastTimestamps[req.ShotID] = req.TsUnixNs

	// Store to database asynchronously
	if s.storage != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.storage.StoreTelemetryPoint(ctx, &req); err != nil {
				log.Printf("failed to store telemetry point: %v", err)
			}
		}()
	}

	resp := telemetry.IngestAck{ShotID: req.ShotID, AcceptedPoints: 1}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *ingestService) close() {
	// Wait for pending storage operations
	s.wg.Wait()

	if s.storage != nil {
		if err := s.storage.Close(); err != nil {
			log.Printf("failed to close storage: %v", err)
		}
	}

	if s.nc != nil && !s.nc.IsClosed() {
		s.nc.Close()
	}
}

func main() {
	cfgPath := flag.String("config", "configs/dev/ingestor.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	svc, err := newIngestService(cfg)
	if err != nil {
		log.Fatalf("connect to nats: %v", err)
	}
	defer svc.close()

	mux := http.NewServeMux()
	mux.Handle("/health", health.Handler("ok"))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/ingest", svc.ingestHandler)

	srv := &http.Server{
		Addr:         cfg.Service.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("ingestor: http listening on %s", cfg.Service.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
