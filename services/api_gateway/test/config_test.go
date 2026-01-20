package test

import (
	"path/filepath"
	"testing"

	"github.com/fusionguard/services/api_gateway/internal/config"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "dev", "api_gateway.yaml")
	if _, err := config.Load(path); err != nil {
		t.Fatalf("load config: %v", err)
	}
}
