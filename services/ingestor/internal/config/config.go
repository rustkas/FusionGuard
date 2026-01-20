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
	GRPCAddr string `yaml:"grpc_addr"`
}

type NATSConfig struct {
	URL        string `yaml:"url"`
	SubjectRaw string `yaml:"subject_raw"`
}

type SamplingConfig struct {
	ResampleHz  int      `yaml:"resample_hz"`
	MaxChannels int      `yaml:"max_channels"`
	Allowed     []string `yaml:"allowed_channels"`
}

type StorageConfig struct {
	PostgresDSN string `yaml:"postgres_dsn"`
	WriteRaw    bool   `yaml:"write_raw"`
}

type Config struct {
	ConfigVersion int            `yaml:"config_version"`
	Service       ServiceConfig  `yaml:"service"`
	NATS          NATSConfig     `yaml:"nats"`
	Sampling      SamplingConfig `yaml:"sampling"`
	Storage       StorageConfig  `yaml:"storage"`
	Timeout       time.Duration  `yaml:"-"`
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

	if cfg.NATS.URL == "" {
		cfg.NATS.URL = "nats://localhost:4222"
	}

	cfg.Timeout = time.Second * 10
	return &cfg, nil
}
