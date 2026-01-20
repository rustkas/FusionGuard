package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fusionguard/pkg/storage"
)

// Mock storage for testing
type mockStorage struct {
	shots      []storage.Shot
	riskSeries storage.RiskSeries
	events     []storage.Event
}

func (m *mockStorage) ListShots() ([]storage.Shot, error) {
	return m.shots, nil
}

func (m *mockStorage) GetRiskSeries(shotID string, fromUnixNs, toUnixNs *int64) (storage.RiskSeries, error) {
	return m.riskSeries, nil
}

func (m *mockStorage) GetTelemetrySeries(shotID string, fromUnixNs, toUnixNs *int64) (storage.TelemetrySeries, error) {
	return storage.TelemetrySeries{}, nil
}

func (m *mockStorage) GetEvents(shotID string) ([]storage.Event, error) {
	return m.events, nil
}

func (m *mockStorage) GetRiskPointAt(shotID string, atUnixNs int64) (storage.RiskPoint, error) {
	return storage.RiskPoint{}, nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestShotsHandler(t *testing.T) {
	mockStore := &mockStorage{
		shots: []storage.Shot{
			{ShotID: "shot1"},
			{ShotID: "shot2"},
		},
	}
	
	api := New(mockStore)
	
	req := httptest.NewRequest("GET", "/shots", nil)
	w := httptest.NewRecorder()
	
	api.shots(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	
	shots, ok := response["shots"].([]interface{})
	if !ok {
		t.Fatal("expected shots array in response")
	}
	
	if len(shots) != 2 {
		t.Errorf("expected 2 shots, got %d", len(shots))
	}
}

func TestSeriesHandler(t *testing.T) {
	mockStore := &mockStorage{
		riskSeries: storage.RiskSeries{
			ShotID: "test_shot",
			Points: []storage.RiskPoint{
				{TsUnixNs: 1000, RiskH50: 0.5, RiskH200: 0.4},
			},
		},
	}
	
	api := New(mockStore)
	
	req := httptest.NewRequest("GET", "/shots/test_shot/series?kind=risk", nil)
	w := httptest.NewRecorder()
	
	api.shotHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestEventsHandler(t *testing.T) {
	mockStore := &mockStorage{
		events: []storage.Event{
			{Kind: "alert", Message: "High risk", TsUnixNs: 1000},
		},
	}
	
	api := New(mockStore)
	
	req := httptest.NewRequest("GET", "/shots/test_shot/events", nil)
	w := httptest.NewRecorder()
	
	api.shotHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
