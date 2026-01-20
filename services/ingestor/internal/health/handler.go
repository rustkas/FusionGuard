package health

import (
	"encoding/json"
	"net/http"
)

// Handler returns a lightweight health check endpoint.
func Handler(status string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
	})
}
