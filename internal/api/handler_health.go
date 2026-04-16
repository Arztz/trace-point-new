package api

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.3",
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Prometheus metrics endpoint
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# Trace-Point metrics\n"))
}

func (s *Server) handleAPIInfo(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"name":    "Trace-Point API",
		"version": "v1.0.3",
		"endpoints": []string{
			"/api/v1/spikes",
			"/api/v1/spikes/:id",
			"/api/v1/spikes/:id/details",
			"/api/v1/spikes/analyze",
			"/api/v1/timeline",
			"/api/v1/export",
			"/api/v1/export/refactoring",
			"/api/v1/config",
			"/api/v1/gravity-scores",
		},
	})
}

// --- Response helpers ---

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
