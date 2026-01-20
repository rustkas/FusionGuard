package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// IngestLatency measures time from receiving telemetry to publishing to NATS
	IngestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_ingestor_latency_seconds",
			Help:    "Latency of telemetry ingestion in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"status"},
	)

	// IngestThroughput counts ingested telemetry points
	IngestThroughput = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_ingestor_points_total",
			Help: "Total number of telemetry points ingested",
		},
		[]string{"shot_id", "status"},
	)

	// StorageLatency measures database write latency
	StorageLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_ingestor_storage_latency_seconds",
			Help:    "Database write latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"operation"},
	)

	// StorageErrors counts database errors
	StorageErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_ingestor_storage_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)
)
