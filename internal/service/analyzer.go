package service

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/integration/prometheus"
)

// Analyzer performs historical spike analysis using sliding window algorithm.
type Analyzer struct {
	cfg        *config.Config
	promClient *prometheus.Client
}

// NewAnalyzer creates a new historical analyzer.
func NewAnalyzer(cfg *config.Config, promClient *prometheus.Client) *Analyzer {
	return &Analyzer{
		cfg:        cfg,
		promClient: promClient,
	}
}

// Analyze performs historical spike analysis over a time range.
func (a *Analyzer) Analyze(req domain.SpikeAnalysisRequest) (*domain.SpikeAnalysisResponse, error) {
	// Set defaults
	if req.Threshold == 0 {
		req.Threshold = a.cfg.Detection.CPUThreshold
	}
	if req.Limit <= 0 {
		req.Limit = 1000
	}

	windowDuration, err := parseWindowDuration(req.Window)
	if err != nil {
		return nil, err
	}

	log.Printf("[Analyzer] Analyzing spikes from %s to %s (window=%s, threshold=%.0f%%)",
		req.Start.Format(time.RFC3339), req.End.Format(time.RFC3339), req.Window, req.Threshold)

	// Fetch timeline metrics from Prometheus
	metrics, err := a.promClient.QueryTimelineMetrics(req.Start, req.End, req.DeploymentName)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline metrics: %w", err)
	}

	if len(metrics) == 0 {
		return &domain.SpikeAnalysisResponse{
			Request: req,
			Summary: domain.SpikeAnalysisSummary{
				TotalSpikes:         0,
				TimeRangeHours:      req.End.Sub(req.Start).Hours(),
				AnalyzedDeployments: 0,
				SpikesByType:        map[string]int{},
			},
			Spikes:     []domain.HistoricalSpike{},
			Pagination: domain.PaginationInfo{Limit: req.Limit, Offset: req.Offset},
		}, nil
	}

	// Group metrics by deployment
	deploymentMetrics := make(map[string][]domain.TimelineMetric)
	for _, m := range metrics {
		if req.Namespace != "" && m.Namespace != req.Namespace {
			continue
		}
		key := m.Namespace + "/" + m.DeploymentName
		deploymentMetrics[key] = append(deploymentMetrics[key], m)
	}

	// Run sliding window analysis per deployment
	var allSpikes []domain.HistoricalSpike
	spikesByType := map[string]int{"cpu": 0, "ram": 0, "both": 0}
	deploymentSpikeCounts := make(map[string]int)

	for deployKey, dMetrics := range deploymentMetrics {
		// Sort by timestamp
		sort.Slice(dMetrics, func(i, j int) bool {
			return dMetrics[i].Timestamp.Before(dMetrics[j].Timestamp)
		})

		spikes := a.detectSpikesInWindow(dMetrics, windowDuration, req.Threshold)

		for _, s := range spikes {
			spikesByType[s.Type]++
			deploymentSpikeCounts[deployKey]++
		}

		allSpikes = append(allSpikes, spikes...)
	}

	// Sort by deviation descending (most severe first)
	sort.Slice(allSpikes, func(i, j int) bool {
		return allSpikes[i].DeviationPercent > allSpikes[j].DeviationPercent
	})

	// Build top deployments
	type dCount struct {
		name  string
		count int
	}
	var topDeploys []dCount
	for k, v := range deploymentSpikeCounts {
		topDeploys = append(topDeploys, dCount{k, v})
	}
	sort.Slice(topDeploys, func(i, j int) bool {
		return topDeploys[i].count > topDeploys[j].count
	})

	topDeployments := make([]domain.DeploymentSpikes, 0)
	for i, d := range topDeploys {
		if i >= 10 {
			break
		}
		topDeployments = append(topDeployments, domain.DeploymentSpikes{
			DeploymentName: d.name,
			SpikeCount:     d.count,
		})
	}

	// Apply pagination
	total := len(allSpikes)
	start := req.Offset
	if start > total {
		start = total
	}
	end := start + req.Limit
	if end > total {
		end = total
	}
	pagedSpikes := allSpikes[start:end]

	return &domain.SpikeAnalysisResponse{
		Request: req,
		Summary: domain.SpikeAnalysisSummary{
			TotalSpikes:         total,
			TimeRangeHours:      req.End.Sub(req.Start).Hours(),
			AnalyzedDeployments: len(deploymentMetrics),
			SpikesByType:        spikesByType,
			TopDeployments:      topDeployments,
		},
		Spikes: pagedSpikes,
		Pagination: domain.PaginationInfo{
			Limit:   req.Limit,
			Offset:  req.Offset,
			HasMore: end < total,
		},
	}, nil
}

func (a *Analyzer) detectSpikesInWindow(metrics []domain.TimelineMetric, windowDuration time.Duration, threshold float64) []domain.HistoricalSpike {
	var spikes []domain.HistoricalSpike

	for i, current := range metrics {
		// Build window: all points from (current - windowDuration) to (current - 1)
		windowStart := current.Timestamp.Add(-windowDuration)
		var cpuSum, ramSum float64
		var count int

		for j := 0; j < i; j++ {
			if metrics[j].Timestamp.After(windowStart) || metrics[j].Timestamp.Equal(windowStart) {
				cpuSum += metrics[j].CPUPercent
				ramSum += metrics[j].RAMPercent
				count++
			}
		}

		if count < 2 { // Need at least 2 points for meaningful average
			continue
		}

		avgCPU := cpuSum / float64(count)
		avgRAM := ramSum / float64(count)

		// Calculate deviation
		var cpuDeviation, ramDeviation float64
		if avgCPU > 0 {
			cpuDeviation = (current.CPUPercent - avgCPU) / avgCPU * 100
		}
		if avgRAM > 0 {
			ramDeviation = (current.RAMPercent - avgRAM) / avgRAM * 100
		}

		cpuSpike := cpuDeviation > threshold
		ramSpike := ramDeviation > threshold

		if !cpuSpike && !ramSpike {
			continue
		}

		spikeType := "cpu"
		deviation := cpuDeviation
		if ramSpike && cpuSpike {
			spikeType = "both"
			if ramDeviation > cpuDeviation {
				deviation = ramDeviation
			}
		} else if ramSpike {
			spikeType = "ram"
			deviation = ramDeviation
		}

		spikes = append(spikes, domain.HistoricalSpike{
			ID:               uuid.New().String(),
			Timestamp:        current.Timestamp,
			DeploymentName:   current.DeploymentName,
			Namespace:        current.Namespace,
			ContainerName:    current.DeploymentName, // Same as deployment for deployment-level
			Type:             spikeType,
			CPUPercent:       current.CPUPercent,
			RAMPercent:       current.RAMPercent,
			MovingAverageCPU: avgCPU,
			MovingAverageRAM: avgRAM,
			ThresholdPercent: threshold,
			DeviationPercent: deviation,
			Severity:         domain.ClassifySeverity(deviation),
		})
	}

	return spikes
}

func parseWindowDuration(window string) (time.Duration, error) {
	switch window {
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return 1 * time.Hour, nil
	case "":
		return 30 * time.Minute, nil // default
	default:
		return 0, fmt.Errorf("invalid window size: %s (valid: 5m, 15m, 30m, 1h)", window)
	}
}
