package api

import (
	"log"
	"net/http"
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

	// Fetch timeline metrics from Prometheus
	metrics, err := s.promClient.QueryTimelineMetrics(start, end, deploymentFilter)
	if err != nil {
		log.Printf("[Timeline] Failed to query metrics: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to query timeline metrics")
		return
	}

	// Get available deployments
	deployments, err := s.promClient.GetAvailableDeployments()
	if err != nil {
		log.Printf("[Timeline] Failed to get available deployments: %v", err)
		// Continue with empty list
		deployments = []domain.AvailableDeployment{}
	}

	// Get spike markers from DB
	spikeEvents, _, err := s.spikeRepo.List(domain.SpikeListFilter{
		Limit: 100,
		Sort:  "time",
		Order: "desc",
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

	// Calculate deployment summaries
	summaries := calculateSummaries(
		metrics,
		s.cfg.Timeline.CPUCloseTo100Threshold, s.cfg.Timeline.CPUFarBelow100Threshold,
		s.cfg.Timeline.RAMCloseTo100Threshold, s.cfg.Timeline.RAMFarBelow100Threshold,
	)

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

func calculateSummaries(metrics []domain.TimelineMetric, cpuHighThreshold, cpuLowThreshold, ramHighThreshold, ramLowThreshold float64) []domain.DeploymentSummary {
	// Group by deployment
	type stats struct {
		totalCPU, totalRAM float64
		maxCPU, maxRAM     float64
		count              int
		namespace          string
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
		s.count++
		if m.CPUPercent > s.maxCPU {
			s.maxCPU = m.CPUPercent
		}
		if m.RAMPercent > s.maxRAM {
			s.maxRAM = m.RAMPercent
		}
	}

	summaries := make([]domain.DeploymentSummary, 0, len(groups))
	for name, s := range groups {
		avgCPU := s.totalCPU / float64(s.count)
		avgRAM := s.totalRAM / float64(s.count)

		summaries = append(summaries, domain.DeploymentSummary{
			DeploymentName:    name,
			Namespace:         s.namespace,
			AvgCPU:            avgCPU,
			MaxCPU:            s.maxCPU,
			AvgRAM:            avgRAM,
			MaxRAM:            s.maxRAM,
			Classification:    domain.ClassifyDeployment(avgCPU, s.maxCPU, cpuHighThreshold, cpuLowThreshold, avgRAM, s.maxRAM, ramHighThreshold, ramLowThreshold),
			CPUClassification: domain.ClassifyResource(avgCPU, s.maxCPU, cpuHighThreshold, cpuLowThreshold),
			RAMClassification: domain.ClassifyResource(avgRAM, s.maxRAM, ramHighThreshold, ramLowThreshold),
		})
	}

	return summaries
}
