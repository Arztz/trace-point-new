package api

import (
	"net/http"
	"time"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	_, dsName, err := s.getInstance(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	events, err := s.spikeRepo.GetAllForExport(dsName, 7)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to export data: "+err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=trace-point-export-"+time.Now().Format("2006-01-02")+".json")
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"exported_at":  time.Now().Format(time.RFC3339),
		"period":       "7 days",
		"total_spikes": len(events),
		"spikes":       events,
	})
}

func (s *Server) handleExportRefactoring(w http.ResponseWriter, r *http.Request) {
	_, dsName, err := s.getInstance(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	report, err := s.gravity.GenerateRefactoringReport(dsName, 7)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate refactoring report: "+err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=refactoring-report-"+time.Now().Format("2006-01-02")+".json")
	respondJSON(w, http.StatusOK, report)
}
