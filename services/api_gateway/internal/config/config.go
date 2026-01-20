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

type StorageConfig struct {
	PostgresDSN string `yaml:"postgres_dsn"`
}

type FeaturesCache struct {
	Enabled    bool `yaml:"enabled"`
	TTLSeconds int  `yaml:"ttl_seconds"`
}

type Config struct {
	ConfigVersion int           `yaml:"config_version"`
	Service       ServiceConfig `yaml:"service"`
	Storage       StorageConfig `yaml:"storage"`
	FeaturesCache FeaturesCache `yaml:"features_cache"`
	Timeout       time.Duration `yaml:"-"`
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

	cfg.Timeout = 10 * time.Second
	if cfg.FeaturesCache.TTLSeconds <= 0 {
		cfg.FeaturesCache.TTLSeconds = 5
	}

	return &cfg, nil
}
