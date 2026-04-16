package api

import (
	"net/http"
	"time"
)

func (s *Server) handleGravityScores(w http.ResponseWriter, r *http.Request) {
	_, dsName, err := s.getInstance(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	scores, err := s.gravity.CalculateScores(dsName, 7)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to calculate gravity scores: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"scores":       scores,
		"generated_at": time.Now().Format(time.RFC3339),
		"period":       "7 days",
	})
}
