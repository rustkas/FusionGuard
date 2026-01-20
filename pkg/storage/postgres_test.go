package storage

import (
	"context"
	"testing"
	"time"
)

// TestStorage requires a running Postgres instance
// Skip if not available
func TestStorageOperations(t *testing.T) {
	// This is a placeholder test that would require a test database
	// In a real scenario, use a testcontainers or in-memory database
	
	dsn := "postgres://fusion:fusion@localhost:5432/fusionguard_test?sslmode=disable"
	store, err := New(dsn)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to database: %v", err)
	}
	defer store.Close()
	
	ctx := context.Background()
	shotID := "test_shot_" + time.Now().Format("20060102150405")
	
	// Test CreateShot
	err = store.CreateShot(ctx, shotID, nil)
	if err != nil {
		t.Fatalf("CreateShot failed: %v", err)
	}
	
	// Test StoreTelemetryPoints
	points := []TelemetryPoint{
		{ShotID: shotID, TsUnixNs: time.Now().UnixNano(), ChannelName: "ip", Value: 1.0, Quality: "good"},
		{ShotID: shotID, TsUnixNs: time.Now().UnixNano() + 1_000_000, ChannelName: "ne", Value: 0.5, Quality: "good"},
	}
	
	err = store.StoreTelemetryPoints(ctx, points)
	if err != nil {
		t.Fatalf("StoreTelemetryPoints failed: %v", err)
	}
	
	// Test StoreRiskPoint
	riskPoint := RiskPoint{
		ShotID:             shotID,
		TsUnixNs:           time.Now().UnixNano(),
		RiskH50:            0.75,
		RiskH200:           0.65,
		ModelVersion:       "test",
		CalibrationVersion: "test",
	}
	
	err = store.StoreRiskPoint(ctx, riskPoint)
	if err != nil {
		t.Fatalf("StoreRiskPoint failed: %v", err)
	}
	
	// Test GetRiskSeries
	series, err := store.GetRiskSeries(ctx, shotID, nil, nil)
	if err != nil {
		t.Fatalf("GetRiskSeries failed: %v", err)
	}
	
	if len(series.Points) == 0 {
		t.Error("expected at least one risk point")
	}
	
	// Test GetTelemetrySeries
	telemetrySeries, err := store.GetTelemetrySeries(ctx, shotID, nil, nil)
	if err != nil {
		t.Fatalf("GetTelemetrySeries failed: %v", err)
	}
	
	if len(telemetrySeries.Channels) == 0 {
		t.Error("expected at least one channel")
	}
	
	// Test ListShots
	shots, err := store.ListShots(ctx)
	if err != nil {
		t.Fatalf("ListShots failed: %v", err)
	}
	
	found := false
	for _, shot := range shots {
		if shot.ShotID == shotID {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("expected to find created shot in list")
	}
}
