package api

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}
	deploymentFilter := r.URL.Query().Get("deployment_name")

	end := time.Now()
	var start time.Time

	switch timeRange {
	case "1h":
		start = end.Add(-1 * time.Hour)
	case "3h":
		start = end.Add(-3 * time.Hour)
	case "5h":
		start = end.Add(-5 * time.Hour)
	case "6h":
		start = end.Add(-6 * time.Hour)
	case "12h":
		start = end.Add(-12 * time.Hour)
	case "1d":
		start = end.Add(-24 * time.Hour)
	case "3d":
		start = end.Add(-3 * 24 * time.Hour)
	case "5d":
		start = end.Add(-5 * 24 * time.Hour)
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
	default:
		start = end.Add(-1 * time.Hour)
	}

	inst, dsName, err := s.getInstance(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Fetch timeline metrics from Prometheus
	log.Println("===========================================")
	log.Println("[Timeline] ========== FETCHING TIMELINE ==========")
	log.Println("===========================================")
	os.Stdout.Sync() // Force flush the log
	metrics, err := inst.PromClient.QueryTimelineMetrics(start, end, deploymentFilter)
	if err != nil {
		log.Printf("[Timeline] Failed to query metrics: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to query timeline metrics")
		return
	}

	// Get available deployments
	deployments, err := inst.PromClient.GetAvailableDeployments()
	if err != nil {
		log.Printf("[Timeline] Failed to get available deployments: %v", err)
		// Continue with empty list
		deployments = []domain.AvailableDeployment{}
	}

	// Get spike markers from DB
	spikeEvents, _, err := s.spikeRepo.List(domain.SpikeListFilter{
		Datasource: dsName,
		Limit:      100,
		Sort:       "time",
		Order:      "desc",
	})
	if err != nil {
		log.Printf("[Timeline] Failed to get spike markers: %v", err)
	}

	// Convert spikes to markers (only those within time range)
	var markers []domain.SpikeMarker
	for _, e := range spikeEvents {
		if e.Timestamp.Before(start) || e.Timestamp.After(end) {
			continue
		}

		deviation := float64(0)
		if e.MovingAveragePercent > 0 {
			deviation = (e.CPUUsagePercent - e.MovingAveragePercent) / e.MovingAveragePercent * 100
		}

		markers = append(markers, domain.SpikeMarker{
			Timestamp:      e.Timestamp,
			DeploymentName: e.DeploymentName,
			Namespace:      e.Namespace,
			CPUPercent:     e.CPUUsagePercent,
			RAMPercent:     e.RAMUsagePercent,
			Severity:       domain.ClassifySeverity(deviation),
			SpikeID:        e.ID,
		})
	}

	// Calculate deployment summaries using LIMIT-based classification
	summaries := calculateSummaries(metrics)

	response := domain.TimelineResponse{
		GeneratedAt:          time.Now(),
		StartDate:            start,
		EndDate:              end,
		Metrics:              metrics,
		SpikeMarkers:         markers,
		AvailableDeployments: deployments,
		Summary:              summaries,
	}

	respondJSON(w, http.StatusOK, response)
}

// calculateSummaries computes deployment summaries with LIMIT-based classification.
// Classification rules (based on LIMIT utilization):
//   - "high": avg_of_limit >= 50%
//   - "low": avg_of_limit <= 10%
//   - "ok": otherwise
func calculateSummaries(metrics []domain.TimelineMetric) []domain.DeploymentSummary {
	// Group by deployment
	type stats struct {
		totalCPU, totalRAM       float64
		totalCPULimit, totalRAMLimit float64
		maxCPU, maxRAM           float64
		maxCPULimit, maxRAMLimit float64
		count                    int
		namespace                string
	}
	groups := make(map[string]*stats)

	for _, m := range metrics {
		key := m.DeploymentName
		if _, ok := groups[key]; !ok {
			groups[key] = &stats{namespace: m.Namespace}
		}
		s := groups[key]
		s.totalCPU += m.CPUPercent
		s.totalRAM += m.RAMPercent
		s.totalCPULimit += m.CPUPercentOfLimit
		s.totalRAMLimit += m.RAMPercentOfLimit
		s.count++
		if m.CPUPercent > s.maxCPU {
			s.maxCPU = m.CPUPercent
		}
		if m.RAMPercent > s.maxRAM {
			s.maxRAM = m.RAMPercent
		}
		if m.CPUPercentOfLimit > s.maxCPULimit {
			s.maxCPULimit = m.CPUPercentOfLimit
		}
		if m.RAMPercentOfLimit > s.maxRAMLimit {
			s.maxRAMLimit = m.RAMPercentOfLimit
		}
	}

	summaries := make([]domain.DeploymentSummary, 0, len(groups))
	for name, s := range groups {
		avgCPU := s.totalCPU / float64(s.count)
		avgRAM := s.totalRAM / float64(s.count)
		avgCPUOfLimit := s.totalCPULimit / float64(s.count)
		avgRAMOfLimit := s.totalRAMLimit / float64(s.count)

		// Use LIMIT-based classification
		cpuClassification := domain.ClassifyResourceOfLimit(avgCPUOfLimit)
		ramClassification := domain.ClassifyResourceOfLimit(avgRAMOfLimit)
		classification := domain.ClassifyDeploymentOfLimit(avgCPUOfLimit, avgRAMOfLimit)

		summaries = append(summaries, domain.DeploymentSummary{
			DeploymentName:    name,
			Namespace:         s.namespace,
			AvgCPU:            avgCPU,
			MaxCPU:            s.maxCPU,
			AvgRAM:            avgRAM,
			MaxRAM:            s.maxRAM,
			AvgCPUOfLimit:     avgCPUOfLimit,
			MaxCPUOfLimit:     s.maxCPULimit,
			AvgRAMOfLimit:     avgRAMOfLimit,
			MaxRAMOfLimit:     s.maxRAMLimit,
			Classification:    classification,
			CPUClassification: cpuClassification,
			RAMClassification: ramClassification,
		})
	}

	return summaries
}
