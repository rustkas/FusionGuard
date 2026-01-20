package storage

import (
	"context"

	"github.com/fusionguard/pkg/storage"
)

// Storage wraps the storage layer for API Gateway
type Storage struct {
	store *storage.Storage
}

// New creates a new Storage instance
func New(postgresDSN string) (*Storage, error) {
	store, err := storage.New(postgresDSN)
	if err != nil {
		return nil, err
	}

	return &Storage{store: store}, nil
}

// Close closes the storage connection
func (s *Storage) Close() error {
	return s.store.Close()
}

// ListShots returns all shots
func (s *Storage) ListShots() ([]storage.Shot, error) {
	return s.store.ListShots(context.Background())
}

// GetRiskSeries returns risk series for a shot
func (s *Storage) GetRiskSeries(shotID string, fromUnixNs, toUnixNs *int64) (storage.RiskSeries, error) {
	return s.store.GetRiskSeries(context.Background(), shotID, fromUnixNs, toUnixNs)
}

// GetTelemetrySeries returns telemetry series for a shot
func (s *Storage) GetTelemetrySeries(shotID string, fromUnixNs, toUnixNs *int64) (storage.TelemetrySeries, error) {
	return s.store.GetTelemetrySeries(context.Background(), shotID, fromUnixNs, toUnixNs)
}

// GetEvents returns events for a shot
func (s *Storage) GetEvents(shotID string) ([]storage.Event, error) {
	return s.store.GetEvents(context.Background(), shotID)
}

// GetRiskPointAt returns a risk point at a specific timestamp
func (s *Storage) GetRiskPointAt(shotID string, atUnixNs int64) (storage.RiskPoint, error) {
	return s.store.GetRiskPointAt(context.Background(), shotID, atUnixNs)
}
