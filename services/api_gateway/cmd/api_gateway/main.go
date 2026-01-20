package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/fusionguard/services/api_gateway/internal/config"
	"github.com/fusionguard/services/api_gateway/internal/health"
	httpapi "github.com/fusionguard/services/api_gateway/internal/http"
	stor "github.com/fusionguard/services/api_gateway/internal/storage"
)

func main() {
	cfgPath := flag.String("config", "configs/dev/api_gateway.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	storage, err := stor.New(cfg.Storage.PostgresDSN)
	if err != nil {
		log.Fatalf("create storage: %v", err)
	}
	defer storage.Close()

	mux := http.NewServeMux()
	mux.Handle("/health", health.Handler("ok"))
	mux.Handle("/metrics", promhttp.Handler())

	// Serve static UI files
	fs := http.FileServer(http.Dir("ui"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "ui/index.html")
		} else {
			fs.ServeHTTP(w, r)
		}
	})

	api := httpapi.New(storage)
	api.Register(mux)

	srv := &http.Server{
		Addr:         cfg.Service.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api gateway listening on %s", cfg.Service.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
