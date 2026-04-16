package api

import (
	"net/http"
)

// handleDatasources returns a list of available datasources to the UI
func (s *Server) handleDatasources(w http.ResponseWriter, r *http.Request) {
	var datasources []map[string]interface{}

	for name, _ := range s.instances {
		// Just returning the name and maybe future metadata
		datasources = append(datasources, map[string]interface{}{
			"id":   name,
			"name": name,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"datasources": datasources,
	})
}
