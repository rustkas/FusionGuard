package processor

import (
	"testing"

	"github.com/fusionguard/pkg/telemetry"
)

func TestChannelWindow(t *testing.T) {
	win := newChannelWindow(5)
	
	// Add values
	win.add(1.0)
	win.add(2.0)
	win.add(3.0)
	
	if len(win.values) != 3 {
		t.Errorf("expected 3 values, got %d", len(win.values))
	}
	
	// Test overflow
	win.add(4.0)
	win.add(5.0)
	win.add(6.0) // Should remove first value
	
	if len(win.values) != 5 {
		t.Errorf("expected 5 values after overflow, got %d", len(win.values))
	}
	
	if win.values[0] != 2.0 {
		t.Errorf("expected first value to be 2.0, got %f", win.values[0])
	}
}

func TestWindowStats(t *testing.T) {
	win := newChannelWindow(10)
	
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		win.add(v)
	}
	
	stats := win.stats(100) // 100ms window
	
	if stats.mean < 2.9 || stats.mean > 3.1 {
		t.Errorf("expected mean ~3.0, got %f", stats.mean)
	}
	
	if stats.min != 1.0 {
		t.Errorf("expected min 1.0, got %f", stats.min)
	}
	
	if stats.max != 5.0 {
		t.Errorf("expected max 5.0, got %f", stats.max)
	}
	
	if stats.last != 5.0 {
		t.Errorf("expected last 5.0, got %f", stats.last)
	}
}

func TestMissingRatio(t *testing.T) {
	fs := &FeatureService{
		expected: map[string]struct{}{
			"ip":      {},
			"ne":      {},
			"dwdt":    {},
			"prad":    {},
			"h_alpha": {},
		},
	}
	
	// Test with all channels present
	point1 := &telemetry.TelemetryPoint{
		Channels: []telemetry.ChannelSample{
			{Name: "ip", Value: 1.0},
			{Name: "ne", Value: 0.5},
			{Name: "dwdt", Value: 0.0},
			{Name: "prad", Value: 0.3},
			{Name: "h_alpha", Value: 0.1},
		},
	}
	
	missing1 := fs.missingRatio(point1)
	if missing1 != 0.0 {
		t.Errorf("expected missing ratio 0.0, got %f", missing1)
	}
	
	// Test with missing channels
	point2 := &telemetry.TelemetryPoint{
		Channels: []telemetry.ChannelSample{
			{Name: "ip", Value: 1.0},
			{Name: "ne", Value: 0.5},
		},
	}
	
	missing2 := fs.missingRatio(point2)
	expected := 3.0 / 5.0 // 3 missing out of 5
	if missing2 != expected {
		t.Errorf("expected missing ratio %f, got %f", expected, missing2)
	}
}
