package recommend

import (
	"testing"
)

func TestRuleEvaluate(t *testing.T) {
	rule := Rule{
		ID: "test_rule",
		When: struct {
			All []Condition `yaml:"all"`
		}{
			All: []Condition{
				{Field: "risk_h50", Op: "gte", Value: 0.8},
				{Field: "risk_h200", Op: "lte", Value: 0.9},
			},
		},
		Then: Recommendation{
			Action:     "reduce_heating",
			Confidence: 0.7,
			Rationale:  "High risk detected",
		},
	}
	
	// Test matching conditions
	features := map[string]float64{}
	risk := map[string]float64{
		"risk_h50":  0.85,
		"risk_h200": 0.8,
	}
	
	if !rule.Evaluate(features, risk) {
		t.Error("expected rule to match")
	}
	
	// Test non-matching conditions
	risk["risk_h50"] = 0.7
	if rule.Evaluate(features, risk) {
		t.Error("expected rule not to match")
	}
}

func TestLookupField(t *testing.T) {
	features := map[string]float64{
		"ip_mean_w50": 1.0,
		"ne_mean_w50": 0.5,
	}
	risk := map[string]float64{
		"risk_h50":  0.8,
		"risk_h200": 0.7,
	}
	
	// Test feature lookup
	val, ok := lookupField("feature.ip_mean_w50", features, risk)
	if !ok || val != 1.0 {
		t.Errorf("expected feature value 1.0, got %f, ok=%v", val, ok)
	}
	
	// Test risk lookup
	val, ok = lookupField("risk_h50", features, risk)
	if !ok || val != 0.8 {
		t.Errorf("expected risk value 0.8, got %f, ok=%v", val, ok)
	}
	
	// Test missing field
	_, ok = lookupField("feature.missing", features, risk)
	if ok {
		t.Error("expected missing field to return false")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		actual float64
		target float64
		op     string
		want   bool
	}{
		{1.0, 0.5, "gte", true},
		{1.0, 0.5, "gt", true},
		{0.5, 1.0, "lte", true},
		{0.5, 1.0, "lt", true},
		{1.0, 1.0, "eq", true},
		{1.0, 2.0, "eq", false},
		{1.0, 0.5, "lte", false},
	}
	
	for _, tt := range tests {
		got := compare(tt.actual, tt.target, tt.op)
		if got != tt.want {
			t.Errorf("compare(%f, %f, %s) = %v, want %v", tt.actual, tt.target, tt.op, got, tt.want)
		}
	}
}
