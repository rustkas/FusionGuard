package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	stor "github.com/fusionguard/services/api_gateway/internal/storage"
)

type API struct {
	storage *stor.Storage
}

func New(storage *stor.Storage) *API {
	return &API{storage: storage}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/shots", a.shots)
	mux.HandleFunc("/shots/", a.shotHandler)
}

func (a *API) shots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shots, err := a.storage.ListShots()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list shots: %v", err), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(shots))
	for _, shot := range shots {
		item := map[string]interface{}{
			"shot_id": shot.ShotID,
		}
		if shot.StartedAt != nil {
			item["started_unix_ns"] = shot.StartedAt.UnixNano()
		}
		if shot.FinishedAt != nil {
			item["finished_unix_ns"] = shot.FinishedAt.UnixNano()
		}
		response = append(response, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"shots": response})
}

func (a *API) shotHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/shots/")
	parts := strings.Split(path, "/")

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "shot_id required", http.StatusBadRequest)
		return
	}

	shotID := parts[0]

	if len(parts) == 1 {
		// This should not happen, but handle it
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	switch parts[1] {
	case "series":
		a.series(w, r, shotID)
	case "events":
		a.events(w, r, shotID)
	case "explain":
		a.explain(w, r, shotID)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (a *API) series(w http.ResponseWriter, r *http.Request, shotID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	kind := r.URL.Query().Get("kind")
	if kind == "" {
		http.Error(w, "kind parameter required", http.StatusBadRequest)
		return
	}

	var fromUnixNs, toUnixNs *int64
	if fromStr := r.URL.Query().Get("from_unix_ns"); fromStr != "" {
		val, err := strconv.ParseInt(fromStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid from_unix_ns", http.StatusBadRequest)
			return
		}
		fromUnixNs = &val
	}
	if toStr := r.URL.Query().Get("to_unix_ns"); toStr != "" {
		val, err := strconv.ParseInt(toStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid to_unix_ns", http.StatusBadRequest)
			return
		}
		toUnixNs = &val
	}

	w.Header().Set("Content-Type", "application/json")

	switch kind {
	case "risk":
		series, err := a.storage.GetRiskSeries(shotID, fromUnixNs, toUnixNs)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get risk series: %v", err), http.StatusInternalServerError)
			return
		}

		points := make([]map[string]interface{}, 0, len(series.Points))
		for _, p := range series.Points {
			points = append(points, map[string]interface{}{
				"ts_unix_ns":         p.TsUnixNs,
				"risk_h50":           p.RiskH50,
				"risk_h200":          p.RiskH200,
				"model_version":      p.ModelVersion,
				"calibration_version": p.CalibrationVersion,
			})
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"shot_id": series.ShotID,
			"points":  points,
		})

	case "telemetry":
		series, err := a.storage.GetTelemetrySeries(shotID, fromUnixNs, toUnixNs)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get telemetry series: %v", err), http.StatusInternalServerError)
			return
		}

		channels := make(map[string][]map[string]interface{})
		for chName, points := range series.Channels {
			chPoints := make([]map[string]interface{}, 0, len(points))
			for _, p := range points {
				chPoints = append(chPoints, map[string]interface{}{
					"ts_unix_ns": p.TsUnixNs,
					"value":      p.Value,
				})
			}
			channels[chName] = chPoints
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"shot_id":  series.ShotID,
			"channels": channels,
		})

	case "features":
		// Features are not stored in DB yet, return empty response
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"shot_id": shotID,
			"vectors": []interface{}{},
		})

	default:
		http.Error(w, "invalid kind, must be risk, telemetry, or features", http.StatusBadRequest)
	}
}

func (a *API) events(w http.ResponseWriter, r *http.Request, shotID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, err := a.storage.GetEvents(shotID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		item := map[string]interface{}{
			"ts_unix_ns": event.TsUnixNs,
			"kind":       event.Kind,
			"message":    event.Message,
		}
		if event.Severity != "" {
			item["severity"] = event.Severity
		}
		response = append(response, item)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"events": response})
}

func (a *API) explain(w http.ResponseWriter, r *http.Request, shotID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	atStr := r.URL.Query().Get("at_unix_ns")
	if atStr == "" {
		http.Error(w, "at_unix_ns parameter required", http.StatusBadRequest)
		return
	}

	atUnixNs, err := strconv.ParseInt(atStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid at_unix_ns", http.StatusBadRequest)
		return
	}

	riskPoint, err := a.storage.GetRiskPointAt(shotID, atUnixNs)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get risk point: %v", err), http.StatusNotFound)
		return
	}

	// For now, we don't store top_features in the database
	// This is a placeholder - in a real implementation, we'd need to store
	// feature contributions or reconstruct them
	topFeatures := []map[string]interface{}{}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ts_unix_ns":  riskPoint.TsUnixNs,
		"top_features": topFeatures,
	})
}
