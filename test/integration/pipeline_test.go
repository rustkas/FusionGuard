package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/fusionguard/pkg/telemetry"
)

// Integration test for end-to-end pipeline
// Requires running services (ingestor, feature_service, inference_service, api_gateway)
// and infrastructure (NATS, Postgres)

const (
	IngestorURL   = "http://localhost:8081"
	APIGatewayURL = "http://localhost:8080"
	TestShotID    = "integration_test_shot"
)

func TestEndToEndPipeline(t *testing.T) {
	// Skip if services are not running
	if !isServiceRunning(IngestorURL) {
		t.Skip("Ingestor service not running, skipping integration test")
	}

	// Step 1: Ingest telemetry point
	point := telemetry.TelemetryPoint{
		ShotID:   TestShotID,
		TsUnixNs: time.Now().UnixNano(),
		Channels: []telemetry.ChannelSample{
			{Name: "ip", Value: 1.0, Quality: telemetry.QualityGood},
			{Name: "ne", Value: 0.5, Quality: telemetry.QualityGood},
			{Name: "dwdt", Value: 0.0, Quality: telemetry.QualityGood},
			{Name: "prad", Value: 0.3, Quality: telemetry.QualityGood},
			{Name: "h_alpha", Value: 0.1, Quality: telemetry.QualityGood},
		},
	}

	pointJSON, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("Failed to marshal telemetry point: %v", err)
	}

	resp, err := http.Post(IngestorURL+"/ingest", "application/json", bytes.NewBuffer(pointJSON))
	if err != nil {
		t.Fatalf("Failed to send telemetry point: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Step 2: Wait for processing (features + inference)
	time.Sleep(2 * time.Second)

	// Step 3: Query API for risk series
	if !isServiceRunning(APIGatewayURL) {
		t.Skip("API Gateway not running, skipping API test")
	}

	riskResp, err := http.Get(APIGatewayURL + "/shots/" + TestShotID + "/series?kind=risk")
	if err != nil {
		t.Fatalf("Failed to query risk series: %v", err)
	}
	defer riskResp.Body.Close()

	if riskResp.StatusCode == http.StatusOK {
		var riskData map[string]interface{}
		if err := json.NewDecoder(riskResp.Body).Decode(&riskData); err != nil {
			t.Logf("Failed to decode risk response: %v", err)
		} else {
			t.Logf("Risk series retrieved: %+v", riskData)
		}
	}

	// Step 4: Query for events
	eventsResp, err := http.Get(APIGatewayURL + "/shots/" + TestShotID + "/events")
	if err != nil {
		t.Fatalf("Failed to query events: %v", err)
	}
	defer eventsResp.Body.Close()

	if eventsResp.StatusCode == http.StatusOK {
		var eventsData map[string]interface{}
		if err := json.NewDecoder(eventsResp.Body).Decode(&eventsData); err != nil {
			t.Logf("Failed to decode events response: %v", err)
		} else {
			t.Logf("Events retrieved: %+v", eventsData)
		}
	}
}

func isServiceRunning(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
