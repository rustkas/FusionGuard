package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// APILatency measures API request latency
	APILatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_api_request_latency_seconds",
			Help:    "API request latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"endpoint", "method", "status"},
	)

	// APIRequests counts API requests
	APIRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"endpoint", "method", "status"},
	)

	// DatabaseQueryLatency measures database query latency
	DatabaseQueryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_api_db_query_latency_seconds",
			Help:    "Database query latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"operation"},
	)
)
