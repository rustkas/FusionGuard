package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FeatureComputationLatency measures time to compute features
	FeatureComputationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_feature_computation_latency_seconds",
			Help:    "Feature computation latency in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05},
		},
		[]string{"window_ms"},
	)

	// FeatureThroughput counts processed feature vectors
	FeatureThroughput = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_feature_vectors_total",
			Help: "Total number of feature vectors computed",
		},
		[]string{"window_ms"},
	)

	// FeatureErrors counts feature computation errors
	FeatureErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_feature_errors_total",
			Help: "Total number of feature computation errors",
		},
		[]string{"error_type"},
	)
)
