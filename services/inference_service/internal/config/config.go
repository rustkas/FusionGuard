package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Name     string `yaml:"name"`
	HTTPAddr string `yaml:"http_addr"`
}

type NATSConfig struct {
	URL             string `yaml:"url"`
	SubjectFeatures string `yaml:"subject_features"`
	SubjectRisk     string `yaml:"subject_risk"`
	SubjectAlerts   string `yaml:"subject_alerts"`
}

type ModelConfig struct {
	Provider         string `yaml:"provider"`
	ModelPath        string `yaml:"model_path"`
	ModelVersion     string `yaml:"model_version"`
	FeatureOrderPath string `yaml:"feature_order_path"`
}

type CalibrationConfig struct {
	Kind               string `yaml:"kind"`
	ParamsPath         string `yaml:"params_path"`
	CalibrationVersion string `yaml:"calibration_version"`
}

type ThresholdConfig struct {
	RiskH50  float64 `yaml:"risk_h50_alert"`
	RiskH200 float64 `yaml:"risk_h200_alert"`
}

type StorageConfig struct {
	PostgresDSN string `yaml:"postgres_dsn"`
	WriteRisk   bool   `yaml:"write_risk"`
}

type RulesConfig struct {
	RulesPath string `yaml:"rules_path"`
}

type Config struct {
	ConfigVersion int               `yaml:"config_version"`
	Service       ServiceConfig     `yaml:"service"`
	NATS          NATSConfig        `yaml:"nats"`
	Model         ModelConfig       `yaml:"model"`
	Calibration   CalibrationConfig `yaml:"calibration"`
	Thresholds    ThresholdConfig   `yaml:"thresholds"`
	Storage       StorageConfig     `yaml:"storage"`
	Rules         RulesConfig       `yaml:"rules"`
	Timeout       time.Duration     `yaml:"-"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Service.HTTPAddr == "" {
		return nil, fmt.Errorf("service.http_addr is required")
	}

	cfg.Timeout = 20 * time.Second
	return &cfg, nil
}
