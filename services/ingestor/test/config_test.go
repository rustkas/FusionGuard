package test

import (
	"path/filepath"
	"testing"

	"github.com/fusionguard/services/ingestor/internal/config"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join("..", "..", "configs", "dev", "ingestor.yaml")

	if _, err := config.Load(path); err != nil {
		t.Fatalf("load config: %v", err)
	}
}
