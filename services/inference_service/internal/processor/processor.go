package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/fusionguard/pkg/storage"
	"github.com/fusionguard/services/inference_service/internal/config"
	modelpkg "github.com/fusionguard/services/inference_service/internal/model"
	"github.com/fusionguard/services/inference_service/internal/recommend"
	stor "github.com/fusionguard/services/inference_service/internal/storage"
)

type FeatureVector struct {
	ShotID       string             `json:"shot_id"`
	TsUnixNs     int64              `json:"ts_unix_ns"`
	WindowMs     int                `json:"window_ms"`
	Features     map[string]float64 `json:"features"`
	MissingRatio float64            `json:"missing_ratio"`
}

type KeyValue struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

type RiskPoint struct {
	ShotID             string                     `json:"shot_id"`
	TsUnixNs           int64                      `json:"ts_unix_ns"`
	RiskH50            float64                    `json:"risk_h50"`
	RiskH200           float64                    `json:"risk_h200"`
	ModelVersion       string                     `json:"model_version"`
	CalibrationVersion string                     `json:"calibration_version"`
	TopFeatures        []KeyValue                 `json:"top_features"`
	Recommendations    []recommend.Recommendation `json:"recommendations"`
}

type Processor struct {
	cfg         *config.Config
	nc          *nats.Conn
	model       *modelpkg.ModelParams
	calibration *modelpkg.Calibration
	rules       []recommend.Rule
	storage     *stor.Storage
	wg          sync.WaitGroup
}

func New(cfg *config.Config) (*Processor, error) {
	nc, err := nats.Connect(cfg.NATS.URL)
	if err != nil {
		return nil, err
	}

	modelParams, err := modelpkg.LoadModel(cfg.Model.ModelPath)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("load model: %w", err)
	}

	calib, err := modelpkg.LoadCalibration(cfg.Calibration.ParamsPath)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("load calibration: %w", err)
	}

	ruleSet, err := recommend.LoadRules(cfg.Rules.RulesPath)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("load rules: %w", err)
	}

	stor, err := stor.New(stor.Config{
		PostgresDSN: cfg.Storage.PostgresDSN,
		WriteRisk:   cfg.Storage.WriteRisk,
		Thresholds: stor.Thresholds{
			RiskH50:  cfg.Thresholds.RiskH50,
			RiskH200: cfg.Thresholds.RiskH200,
		},
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create storage: %w", err)
	}

	return &Processor{
		cfg:         cfg,
		nc:          nc,
		model:       modelParams,
		calibration: calib,
		rules:       ruleSet,
		storage:     stor,
	}, nil
}

func (p *Processor) Close() {
	// Wait for pending storage operations
	p.wg.Wait()

	if p.storage != nil {
		if err := p.storage.Close(); err != nil {
			log.Printf("failed to close storage: %v", err)
		}
	}

	if p.nc != nil && !p.nc.IsClosed() {
		p.nc.Drain()
		p.nc.Close()
	}
}

func (p *Processor) Start(ctx context.Context) error {
	_, err := p.nc.QueueSubscribe(p.cfg.NATS.SubjectFeatures, "inference-service", p.handleVector)
	if err != nil {
		return err
	}

	if err := p.nc.Flush(); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		p.Close()
	}()

	return nil
}

func (p *Processor) handleVector(msg *nats.Msg) {
	var vector FeatureVector
	if err := json.Unmarshal(msg.Data, &vector); err != nil {
		log.Printf("decode feature vector: %v", err)
		return
	}

	score := p.model.Score(vector.Features)
	calibrated := p.calibration.Apply(score)
	riskH50 := calibrated
	riskH200 := math.Min(1.0, calibrated*0.9+0.05)

	riskMap := map[string]float64{"risk_h50": riskH50, "risk_h200": riskH200}

	rp := RiskPoint{
		ShotID:             vector.ShotID,
		TsUnixNs:           vector.TsUnixNs,
		RiskH50:            riskH50,
		RiskH200:           riskH200,
		ModelVersion:       p.model.Version,
		CalibrationVersion: p.calibration.Version,
		TopFeatures:        p.topFeatures(vector.Features),
		Recommendations:    p.recommend(vector.Features, riskMap),
	}

	payload, err := json.Marshal(rp)
	if err != nil {
		return
	}

	if err := p.nc.Publish(p.cfg.NATS.SubjectRisk, payload); err != nil {
		log.Printf("publish risk: %v", err)
	}

	// Store to database asynchronously
	if p.storage != nil {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			storageRP := storage.RiskPoint{
				ShotID:             rp.ShotID,
				TsUnixNs:           rp.TsUnixNs,
				RiskH50:            rp.RiskH50,
				RiskH200:           rp.RiskH200,
				ModelVersion:       rp.ModelVersion,
				CalibrationVersion: rp.CalibrationVersion,
			}

			if err := p.storage.StoreRiskPoint(ctx, storageRP); err != nil {
				log.Printf("failed to store risk point: %v", err)
			}
		}()
	}
}

func (p *Processor) topFeatures(features map[string]float64) []KeyValue {
	type entry struct {
		name  string
		score float64
		absScore float64
	}

	entries := make([]entry, 0, len(features))
	
	// Compute contributions for each feature
	// For linear models: contribution = feature_value * coefficient
	// We also consider the magnitude relative to typical values
	for name, value := range features {
		coef, hasCoef := p.model.Coefficients[name]
		if !hasCoef {
			// Feature not in model, skip
			continue
		}
		
		// Contribution to the score
		contribution := value * coef
		
		// Normalize by feature magnitude for better interpretability
		// This gives us a sense of "how much does this feature contribute relative to its scale"
		normalizedContribution := contribution
		if math.Abs(value) > 1e-9 {
			// Relative contribution (percentage-like)
			normalizedContribution = contribution / math.Abs(value) * 100.0
		}
		
		entries = append(entries, entry{
			name:     name,
			score:    contribution,
			absScore: math.Abs(contribution),
		})
	}

	// Sort by absolute contribution
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].absScore > entries[j].absScore
	})

	// Return top 5 features (increased from 3 for better explainability)
	limit := 5
	if len(entries) < limit {
		limit = len(entries)
	}

	result := make([]KeyValue, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, KeyValue{
			Key:   entries[i].name,
			Value: entries[i].score,
		})
	}

	return result
}

func (p *Processor) recommend(features map[string]float64, risk map[string]float64) []recommend.Recommendation {
	recs := make([]recommend.Recommendation, 0, len(p.rules))
	for _, rule := range p.rules {
		if rule.Evaluate(features, risk) {
			recs = append(recs, rule.Then)
		}
	}
	return recs
}
