package storage

import (
	"time"

	"github.com/fusionguard/pkg/telemetry"
)

// Shot represents a plasma discharge
type Shot struct {
	ShotID      string
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// TelemetryPoint represents a single telemetry measurement
type TelemetryPoint struct {
	ShotID     string
	TsUnixNs   int64
	ChannelName string
	Value      float64
	Quality    string
}

// RiskPoint represents a disruption risk prediction
type RiskPoint struct {
	ShotID             string
	TsUnixNs           int64
	RiskH50            float64
	RiskH200           float64
	ModelVersion       string
	CalibrationVersion string
}

// Event represents an alert or disruption event
type Event struct {
	ID        int64
	ShotID    string
	TsUnixNs  int64
	Kind      string // "alert", "disruption", "info"
	Message   string
	Severity  string // "low", "medium", "high"
	CreatedAt time.Time
}

// TelemetrySeries represents a time series of telemetry for a shot
type TelemetrySeries struct {
	ShotID   string
	Channels map[string][]TelemetryChannelPoint
}

// TelemetryChannelPoint represents a single point in a channel time series
type TelemetryChannelPoint struct {
	TsUnixNs int64
	Value     float64
}

// RiskSeries represents a time series of risk predictions
type RiskSeries struct {
	ShotID string
	Points []RiskPoint
}

// ConvertTelemetryPoint converts pkg/telemetry.TelemetryPoint to storage format
func ConvertTelemetryPoint(tp *telemetry.TelemetryPoint) []TelemetryPoint {
	result := make([]TelemetryPoint, 0, len(tp.Channels))
	for _, ch := range tp.Channels {
		result = append(result, TelemetryPoint{
			ShotID:      tp.ShotID,
			TsUnixNs:    tp.TsUnixNs,
			ChannelName: ch.Name,
			Value:       ch.Value,
			Quality:     string(ch.Quality),
		})
	}
	return result
}
