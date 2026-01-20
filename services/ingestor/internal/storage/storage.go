package storage

import (
	"context"
	"log"
	"time"

	"github.com/fusionguard/pkg/storage"
	"github.com/fusionguard/pkg/telemetry"
)

// Storage wraps the storage layer with ingestor-specific logic
type Storage struct {
	store *storage.Storage
	cfg   Config
}

// Config for storage operations
type Config struct {
	PostgresDSN string
	WriteRaw    bool
}

// New creates a new Storage instance
func New(cfg Config) (*Storage, error) {
	if !cfg.WriteRaw {
		return &Storage{store: nil, cfg: cfg}, nil
	}

	store, err := storage.New(cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	return &Storage{store: store, cfg: cfg}, nil
}

// Close closes the storage connection
func (s *Storage) Close() error {
	if s.store == nil {
		return nil
	}
	return s.store.Close()
}

// StoreTelemetryPoint stores a telemetry point and creates/updates shot record
func (s *Storage) StoreTelemetryPoint(ctx context.Context, tp *telemetry.TelemetryPoint) error {
	if s.store == nil || !s.cfg.WriteRaw {
		return nil
	}

	// Create or update shot record
	now := time.Now()
	if err := s.store.CreateShot(ctx, tp.ShotID, &now); err != nil {
		log.Printf("failed to create shot %s: %v", tp.ShotID, err)
		// Don't fail the whole operation if shot creation fails
	}

	// Store telemetry points
	points := storage.ConvertTelemetryPoint(tp)
	if err := s.store.StoreTelemetryPoints(ctx, points); err != nil {
		return err
	}

	return nil
}
