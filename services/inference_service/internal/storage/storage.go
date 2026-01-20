package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/fusionguard/pkg/storage"
)

// Storage wraps the storage layer with inference-specific logic
type Storage struct {
	store *storage.Storage
	cfg   Config
}

// Config for storage operations
type Config struct {
	PostgresDSN string
	WriteRisk   bool
	Thresholds  Thresholds
}

// Thresholds for alert generation
type Thresholds struct {
	RiskH50  float64
	RiskH200 float64
}

// New creates a new Storage instance
func New(cfg Config) (*Storage, error) {
	if !cfg.WriteRisk {
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

// StoreRiskPoint stores a risk point and creates events if thresholds are exceeded
func (s *Storage) StoreRiskPoint(ctx context.Context, rp storage.RiskPoint) error {
	if s.store == nil || !s.cfg.WriteRisk {
		return nil
	}

	// Store risk point
	if err := s.store.StoreRiskPoint(ctx, rp); err != nil {
		return err
	}

	// Check thresholds and create events
	if rp.RiskH50 >= s.cfg.Thresholds.RiskH50 {
		severity := "medium"
		if rp.RiskH50 >= 0.9 {
			severity = "high"
		}
		event := storage.Event{
			ShotID:   rp.ShotID,
			TsUnixNs: rp.TsUnixNs,
			Kind:     "alert",
			Message:  fmt.Sprintf("High disruption risk (h50=%.3f)", rp.RiskH50),
			Severity: severity,
		}
		if _, err := s.store.CreateEvent(ctx, event); err != nil {
			log.Printf("failed to create alert event: %v", err)
		}
	}

	if rp.RiskH200 >= s.cfg.Thresholds.RiskH200 {
		severity := "medium"
		if rp.RiskH200 >= 0.9 {
			severity = "high"
		}
		event := storage.Event{
			ShotID:   rp.ShotID,
			TsUnixNs: rp.TsUnixNs,
			Kind:     "alert",
			Message:  fmt.Sprintf("High disruption risk (h200=%.3f)", rp.RiskH200),
			Severity: severity,
		}
		if _, err := s.store.CreateEvent(ctx, event); err != nil {
			log.Printf("failed to create alert event: %v", err)
		}
	}

	return nil
}
