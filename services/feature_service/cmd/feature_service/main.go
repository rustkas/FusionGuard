package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fusionguard/services/feature_service/internal/config"
	"github.com/fusionguard/services/feature_service/internal/health"
	"github.com/fusionguard/services/feature_service/internal/processor"
)

func main() {
	cfgPath := flag.String("config", "configs/dev/features.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := processor.New(cfg)
	if err != nil {
		log.Fatalf("feature processor: %v", err)
	}
	defer svc.Close()

	if err := svc.Start(ctx); err != nil {
		log.Fatalf("start processor: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/health", health.Handler("ok"))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/windows", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg.Windows)
	})

	srv := &http.Server{
		Addr:         cfg.Service.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("feature service listening on %s", cfg.Service.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
