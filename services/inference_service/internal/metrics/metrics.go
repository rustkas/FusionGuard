package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// InferenceLatency measures time from feature vector to risk prediction
	InferenceLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_inference_latency_seconds",
			Help:    "Inference latency in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05},
		},
		[]string{"horizon"},
	)

	// RiskPredictions counts risk predictions made
	RiskPredictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_risk_predictions_total",
			Help: "Total number of risk predictions",
		},
		[]string{"horizon", "risk_level"},
	)

	// ModelVersion exposes current model version
	ModelVersion = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fusionguard_model_version",
			Help: "Model version (1 = loaded, 0 = not loaded)",
		},
		[]string{"model_version", "calibration_version"},
	)

	// StorageLatency measures database write latency for risks
	StorageLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fusionguard_inference_storage_latency_seconds",
			Help:    "Database write latency for risk points in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"operation"},
	)

	// AlertCount counts alerts generated
	AlertCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fusionguard_alerts_total",
			Help: "Total number of alerts generated",
		},
		[]string{"severity", "horizon"},
	)
)
