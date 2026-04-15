package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/trace-point/trace-point-renew/internal/domain"
)

func (s *Server) handleListSpikes(w http.ResponseWriter, r *http.Request) {
	filter := domain.SpikeListFilter{
		Namespace:      r.URL.Query().Get("namespace"),
		DeploymentName: r.URL.Query().Get("deployment"),
		Sort:           r.URL.Query().Get("sort"),
		Order:          r.URL.Query().Get("order"),
	}

	if limit, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		filter.Limit = limit
	} else {
		filter.Limit = 50
	}

	if offset, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
		filter.Offset = offset
	}

	events, total, err := s.spikeRepo.List(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list spikes: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"spikes": events,
		"total":  total,
		"pagination": domain.PaginationInfo{
			Limit:   filter.Limit,
			Offset:  filter.Offset,
			HasMore: filter.Offset+filter.Limit < total,
		},
	})
}

func (s *Server) handleGetSpike(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "spike id is required")
		return
	}

	event, err := s.spikeRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get spike: "+err.Error())
		return
	}
	if event == nil {
		respondError(w, http.StatusNotFound, "spike not found")
		return
	}

	respondJSON(w, http.StatusOK, event)
}

func (s *Server) handleGetSpikeDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "spike id is required")
		return
	}

	event, err := s.spikeRepo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get spike details: "+err.Error())
		return
	}
	if event == nil {
		respondError(w, http.StatusNotFound, "spike not found")
		return
	}

	detail := domain.SpikeDetail{
		SpikeEvent: *event,
	}

	respondJSON(w, http.StatusOK, detail)
}

func (s *Server) handleRetrySpikeCorrelation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "spike id is required")
		return
	}

	event, err := s.correlator.RetryCorrelation(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to retry correlation: "+err.Error())
		return
	}
	if event == nil {
		respondError(w, http.StatusNotFound, "spike not found")
		return
	}

	respondJSON(w, http.StatusOK, event)
}
