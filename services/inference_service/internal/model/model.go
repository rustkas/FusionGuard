package model

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type ModelParams struct {
	Coefficients map[string]float64 `json:"coefficients"`
	Intercept    float64            `json:"intercept"`
	Version      string             `json:"version"`
}

type Calibration struct {
	Scale   float64 `json:"scale"`
	Offset  float64 `json:"offset"`
	Version string  `json:"version"`
}

func LoadModel(path string) (*ModelParams, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read model: %w", err)
	}

	var params ModelParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("parse model: %w", err)
	}

	if params.Coefficients == nil {
		params.Coefficients = map[string]float64{}
	}
	return &params, nil
}

func LoadCalibration(path string) (*Calibration, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read calibration: %w", err)
	}

	var calib Calibration
	if err := json.Unmarshal(payload, &calib); err != nil {
		return nil, fmt.Errorf("parse calibration: %w", err)
	}

	if calib.Scale == 0 {
		calib.Scale = 1
	}

	return &calib, nil
}

func (m *ModelParams) Score(features map[string]float64) float64 {
	score := m.Intercept
	for name, value := range features {
		score += value * m.Coefficients[name]
	}
	return score
}

func (c *Calibration) Apply(score float64) float64 {
	adjusted := score*c.Scale + c.Offset
	return 1 / (1 + math.Exp(-adjusted))
}
