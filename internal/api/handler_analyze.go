package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

func (s *Server) handleAnalyzeSpikes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Parse time range
	var start, end time.Time
	var err error

	if startStr := q.Get("start"); startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid start time format (use RFC3339)")
			return
		}
	} else {
		start = time.Now().Add(-24 * time.Hour)
	}

	if endStr := q.Get("end"); endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid end time format (use RFC3339)")
			return
		}
	} else {
		end = time.Now()
	}

	// Parse other parameters
	window := q.Get("window")
	if window == "" {
		window = "30m"
	}

	threshold := s.cfg.Detection.CPUThreshold
	if thresholdStr := q.Get("threshold"); thresholdStr != "" {
		if t, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = t
		}
	}

	limit := 1000
	if limitStr := q.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	offset := 0
	if offsetStr := q.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	req := domain.SpikeAnalysisRequest{
		Start:          start,
		End:            end,
		Window:         window,
		Datasource:     q.Get("datasource"),
		Namespace:      q.Get("namespace"),
		DeploymentName: q.Get("deployment"),
		Threshold:      threshold,
		Limit:          limit,
		Offset:         offset,
	}

	inst, _, err := s.getInstance(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := inst.Analyzer.Analyze(req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Analysis failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}
