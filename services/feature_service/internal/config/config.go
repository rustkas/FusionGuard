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
	URL string `yaml:"url"`
}

type PerChannelConfig struct {
	Stats  []string `yaml:"stats"`
	Zscore struct {
		Enabled  bool `yaml:"enabled"`
		Baseline struct {
			Kind  string  `yaml:"kind"`
			Alpha float64 `yaml:"alpha"`
		} `yaml:"baseline"`
	} `yaml:"zscore"`
}

type GlobalConfig struct {
	IncludeMissing bool    `yaml:"include_missing_ratio"`
	MaxMissing     float64 `yaml:"max_missing_ratio"`
}

type Config struct {
	ConfigVersion int              `yaml:"config_version"`
	Service       ServiceConfig    `yaml:"service"`
	NATS          NATSConfig       `yaml:"nats"`
	Windows       []int            `yaml:"windows_ms"`
	PerChannel    PerChannelConfig `yaml:"per_channel"`
	Global        GlobalConfig     `yaml:"global"`
	Channels      []string         `yaml:"channels"`
	Timeout       time.Duration    `yaml:"-"`
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

	if len(cfg.Channels) == 0 {
		return nil, fmt.Errorf("channels is required")
	}

	cfg.Timeout = 15 * time.Second
	return &cfg, nil
}
