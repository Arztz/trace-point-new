package domain

import "time"

// SpikeEvent represents a detected resource spike for a deployment.
type SpikeEvent struct {
	ID                   string     `json:"id"`
	Timestamp            time.Time  `json:"timestamp"`
	DeploymentName       string     `json:"deployment_name"`
	Namespace            string     `json:"namespace"`
	CPUUsagePercent      float64    `json:"cpu_usage_percent"`
	CPULimitPercent      float64    `json:"cpu_limit_percent"`
	RAMUsagePercent      float64    `json:"ram_usage_percent"`
	RAMLimitPercent      float64    `json:"ram_limit_percent"`
	ThresholdPercent     float64    `json:"threshold_percent"`
	MovingAveragePercent float64    `json:"moving_average_percent"`
	RouteName            *string    `json:"route_name,omitempty"`
	TraceID              *string    `json:"trace_id,omitempty"`
	CulpritFunction      *string    `json:"culprit_function,omitempty"`
	CulpritFilePath      *string    `json:"culprit_file_path,omitempty"`
	AlertSent            bool       `json:"alert_sent"`
	CooldownEnd          *time.Time `json:"cooldown_end,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

// SpikeDetail contains the full details of a spike event including correlation data.
type SpikeDetail struct {
	SpikeEvent
	Correlation *CorrelationResult `json:"correlation,omitempty"`
}

// HistoricalSpike represents a spike detected during historical analysis.
type HistoricalSpike struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	DeploymentName   string    `json:"deployment_name"`
	Namespace        string    `json:"namespace"`
	ContainerName    string    `json:"container_name"`
	Type             string    `json:"type"` // "cpu", "ram", "both"
	CPUPercent       float64   `json:"cpu_percent"`
	RAMPercent       float64   `json:"ram_percent"`
	MovingAverageCPU float64   `json:"moving_average_cpu"`
	MovingAverageRAM float64   `json:"moving_average_ram"`
	ThresholdPercent float64   `json:"threshold_percent"`
	DeviationPercent float64   `json:"deviation_percent"`
	Severity         string    `json:"severity"` // "critical", "medium", "low"
}

// SpikeListFilter holds query parameters for listing spikes.
type SpikeListFilter struct {
	Namespace      string `json:"namespace"`
	DeploymentName string `json:"deployment_name"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
	Sort           string `json:"sort"`  // "time", "cpu", "ram", "deployment"
	Order          string `json:"order"` // "asc", "desc"
}

// SpikeAnalysisRequest holds parameters for historical spike analysis.
type SpikeAnalysisRequest struct {
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	Window         string    `json:"window"` // "5m", "15m", "30m", "1h"
	Namespace      string    `json:"namespace"`
	DeploymentName string    `json:"deployment_name"`
	Threshold      float64   `json:"threshold"`
	Limit          int       `json:"limit"`
	Offset         int       `json:"offset"`
}

// SpikeAnalysisResponse holds the results of historical spike analysis.
type SpikeAnalysisResponse struct {
	Request    SpikeAnalysisRequest    `json:"request"`
	Summary    SpikeAnalysisSummary    `json:"summary"`
	Spikes     []HistoricalSpike       `json:"spikes"`
	Pagination PaginationInfo          `json:"pagination"`
}

// SpikeAnalysisSummary provides aggregate stats for a spike analysis run.
type SpikeAnalysisSummary struct {
	TotalSpikes          int                `json:"total_spikes"`
	TimeRangeHours       float64            `json:"time_range_hours"`
	AnalyzedDeployments  int                `json:"analyzed_deployments"`
	SpikesByType         map[string]int     `json:"spikes_by_type"`
	TopDeployments       []DeploymentSpikes `json:"top_deployments"`
}

// DeploymentSpikes maps a deployment to its spike count.
type DeploymentSpikes struct {
	DeploymentName string `json:"deployment_name"`
	SpikeCount     int    `json:"spike_count"`
}

// PaginationInfo holds pagination metadata.
type PaginationInfo struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// ClassifySeverity returns the severity based on deviation percentage.
func ClassifySeverity(deviationPercent float64) string {
	switch {
	case deviationPercent > 200:
		return "critical"
	case deviationPercent > 100:
		return "medium"
	case deviationPercent > 50:
		return "low"
	default:
		return "normal"
	}
}
