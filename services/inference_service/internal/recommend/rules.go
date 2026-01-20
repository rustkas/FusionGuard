package recommend

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type RuleSet struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	ID   string `yaml:"id"`
	When struct {
		All []Condition `yaml:"all"`
	} `yaml:"when"`
	Then Recommendation `yaml:"then"`
}

type Condition struct {
	Field string  `yaml:"field"`
	Op    string  `yaml:"op"`
	Value float64 `yaml:"value"`
}

type Recommendation struct {
	Action     string  `yaml:"action"`
	Confidence float64 `yaml:"confidence"`
	Rationale  string  `yaml:"rationale"`
}

func LoadRules(path string) ([]Rule, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rules: %w", err)
	}

	var set RuleSet
	if err := yaml.Unmarshal(payload, &set); err != nil {
		return nil, fmt.Errorf("parse rules: %w", err)
	}
	return set.Rules, nil
}

func (r Rule) Evaluate(features map[string]float64, risk map[string]float64) bool {
	for _, cond := range r.When.All {
		val, ok := lookupField(cond.Field, features, risk)
		if !ok {
			return false
		}
		if !compare(val, cond.Value, cond.Op) {
			return false
		}
	}
	return true
}

func lookupField(field string, features map[string]float64, risk map[string]float64) (float64, bool) {
	switch {
	case strings.HasPrefix(field, "feature."):
		key := strings.TrimPrefix(field, "feature.")
		v, ok := features[key]
		return v, ok
	default:
		v, ok := risk[field]
		return v, ok
	}
}

func compare(actual, target float64, op string) bool {
	switch strings.ToLower(op) {
	case "gte":
		return actual >= target
	case "lte":
		return actual <= target
	case "gt":
		return actual > target
	case "lt":
		return actual < target
	case "eq":
		return actual == target
	default:
		return false
	}
}
