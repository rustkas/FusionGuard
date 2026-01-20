package test

import (
	"path/filepath"
	"testing"

	"github.com/fusionguard/services/inference_service/internal/config"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "dev", "inference_service.yaml")
	if _, err := config.Load(path); err != nil {
		t.Fatalf("load config: %v", err)
	}
}
