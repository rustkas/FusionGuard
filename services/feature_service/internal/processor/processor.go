package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/nats-io/nats.go"

	"github.com/fusionguard/pkg/telemetry"
	"github.com/fusionguard/services/feature_service/internal/config"
)

type FeatureVector struct {
	ShotID       string             `json:"shot_id"`
	TsUnixNs     int64              `json:"ts_unix_ns"`
	WindowMs     int                `json:"window_ms"`
	Features     map[string]float64 `json:"features"`
	MissingRatio float64            `json:"missing_ratio"`
}

type FeatureService struct {
	cfg      *config.Config
	nc       *nats.Conn
	windows  map[int]map[string]*channelWindow
	mu       sync.Mutex
	expected map[string]struct{}
}

type channelWindow struct {
	capacity  int
	values    []float64
	lastValue float64
	prevValue float64
	hasPrev   bool
}

type windowStats struct {
	mean  float64
	std   float64
	min   float64
	max   float64
	slope float64
	last  float64
	delta float64
}

func newChannelWindow(capacity int) *channelWindow {
	return &channelWindow{capacity: capacity}
}

func (w *channelWindow) add(value float64) {
	if len(w.values) > 0 {
		w.prevValue = w.lastValue
		w.hasPrev = true
	}

	w.lastValue = value
	w.values = append(w.values, value)
	if len(w.values) > w.capacity {
		w.values = w.values[1:]
	}
}

func (w *channelWindow) stats(windowMs int) windowStats {
	n := len(w.values)
	if n == 0 {
		return windowStats{}
	}

	var sum float64
	minVal := math.Inf(1)
	maxVal := math.Inf(-1)
	for _, v := range w.values {
		sum += v
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	mean := sum / float64(n)
	var variance float64
	for _, v := range w.values {
		delta := v - mean
		variance += delta * delta
	}

	std := 0.0
	if n > 1 {
		std = math.Sqrt(variance / float64(n-1))
	}

	slope := 0.0
	if n >= 2 && windowMs > 0 {
		slope = (w.lastValue - w.values[0]) / float64(windowMs)
	}

	delta := 0.0
	if w.hasPrev {
		delta = w.lastValue - w.prevValue
	}

	return windowStats{mean: mean, std: std, min: minVal, max: maxVal, slope: slope, last: w.lastValue, delta: delta}
}

func New(cfg *config.Config) (*FeatureService, error) {
	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, err
	}

	windows := map[int]map[string]*channelWindow{}
	for _, w := range cfg.Windows {
		windows[w] = map[string]*channelWindow{}
	}

	expected := map[string]struct{}{}
	for _, ch := range cfg.Channels {
		expected[ch] = struct{}{}
	}

	return &FeatureService{
		cfg:      cfg,
		nc:       nc,
		windows:  windows,
		expected: expected,
	}, nil
}

func (s *FeatureService) Close() {
	if s.nc != nil && !s.nc.IsClosed() {
		s.nc.Drain()
		s.nc.Close()
	}
}

func (s *FeatureService) Start(ctx context.Context) error {
	_, err := s.nc.QueueSubscribe(s.cfg.NATS.SubjectRaw, "feature-service", s.handleRaw)
	if err != nil {
		return err
	}

	if err := s.nc.Flush(); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	return nil
}

func (s *FeatureService) handleRaw(msg *nats.Msg) {
	var point telemetry.TelemetryPoint
	if err := json.Unmarshal(msg.Data, &point); err != nil {
		return
	}

	vectors := s.buildFeatureVectors(&point)
	for _, vector := range vectors {
		payload, err := json.Marshal(vector)
		if err != nil {
			continue
		}
		_ = s.nc.Publish(s.cfg.NATS.SubjectFeatures, payload)
	}
}

func (s *FeatureService) buildFeatureVectors(point *telemetry.TelemetryPoint) []FeatureVector {
	missing := s.missingRatio(point)
	vectors := make([]FeatureVector, 0, len(s.cfg.Windows))

	for _, window := range s.cfg.Windows {
		features := s.collectFeatures(point, window, missing)
		vectors = append(vectors, FeatureVector{
			ShotID:       point.ShotID,
			TsUnixNs:     point.TsUnixNs,
			WindowMs:     window,
			Features:     features,
			MissingRatio: missing,
		})
	}

	return vectors
}

func (s *FeatureService) collectFeatures(point *telemetry.TelemetryPoint, window int, missing float64) map[string]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	buffer := s.windows[window]
	features := make(map[string]float64, len(point.Channels)*7+1)

	for _, ch := range point.Channels {
		win := buffer[ch.Name]
		if win == nil {
			win = newChannelWindow(window)
			buffer[ch.Name] = win
		}
		win.add(ch.Value)
		stats := win.stats(window)
		features[fmt.Sprintf("%s_mean_w%d", ch.Name, window)] = stats.mean
		features[fmt.Sprintf("%s_std_w%d", ch.Name, window)] = stats.std
		features[fmt.Sprintf("%s_min_w%d", ch.Name, window)] = stats.min
		features[fmt.Sprintf("%s_max_w%d", ch.Name, window)] = stats.max
		features[fmt.Sprintf("%s_slope_w%d", ch.Name, window)] = stats.slope
		features[fmt.Sprintf("%s_last_w%d", ch.Name, window)] = stats.last
		features[fmt.Sprintf("%s_delta_w%d", ch.Name, window)] = stats.delta
	}

	features["missing_ratio"] = missing
	return features
}

func (s *FeatureService) missingRatio(point *telemetry.TelemetryPoint) float64 {
	seen := map[string]struct{}{}
	for _, ch := range point.Channels {
		seen[ch.Name] = struct{}{}
	}

	missing := len(s.expected) - len(seen)
	if missing < 0 {
		missing = 0
	}

	if len(s.expected) == 0 {
		return 0
	}

	return float64(missing) / float64(len(s.expected))
}
