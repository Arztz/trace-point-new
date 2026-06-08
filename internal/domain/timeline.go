package domain

import "time"

// TimelineMetric represents a single data point in the timeline visualization.
type TimelineMetric struct {
	Timestamp          time.Time `json:"timestamp"`
	DeploymentName     string    `json:"deployment_name"`
	Namespace          string    `json:"namespace"`
	CPUPercent         float64   `json:"cpu_percent"`          // % of request
	RAMPercent         float64   `json:"ram_percent"`          // % of request
	CPUPercentOfLimit  float64   `json:"cpu_percent_of_limit"` // % of limit
	RAMPercentOfLimit  float64   `json:"ram_percent_of_limit"` // % of limit
}

// TimelineResponse is the complete response for the timeline API.
type TimelineResponse struct {
	GeneratedAt          time.Time             `json:"generated_at"`
	StartDate            time.Time             `json:"start_date"`
	EndDate              time.Time             `json:"end_date"`
	Metrics              []TimelineMetric      `json:"metrics"`
	SpikeMarkers         []SpikeMarker         `json:"spike_markers"`
	AvailableDeployments []AvailableDeployment `json:"available_deployments"`
	Summary              []DeploymentSummary   `json:"summary"`
}

// SpikeMarker represents a visual marker on the timeline for a spike event.
type SpikeMarker struct {
	Timestamp      time.Time `json:"timestamp"`
	DeploymentName string    `json:"deployment_name"`
	Namespace      string    `json:"namespace"`
	CPUPercent     float64   `json:"cpu_percent"`
	RAMPercent     float64   `json:"ram_percent"`
	Severity       string    `json:"severity"`
	SpikeID        string    `json:"spike_id,omitempty"`
}

// AvailableDeployment represents a deployment visible in the dropdown selector.
type AvailableDeployment struct {
	Name       string  `json:"name"`
	Namespace  string  `json:"namespace"`
	CurrentCPU float64 `json:"current_cpu"`
	CurrentRAM float64 `json:"current_ram"`
}

// DeploymentSummary provides classification summary for a deployment.
type DeploymentSummary struct {
	DeploymentName     string  `json:"deployment_name"`
	Namespace          string  `json:"namespace"`
	AvgCPU             float64 `json:"avg_cpu"`              // avg % of request
	MaxCPU             float64 `json:"max_cpu"`              // max % of request
	AvgRAM             float64 `json:"avg_ram"`              // avg % of request
	MaxRAM             float64 `json:"max_ram"`              // max % of request
	AvgCPUOfLimit      float64 `json:"avg_cpu_of_limit"`     // avg % of limit (for classification)
	MaxCPUOfLimit      float64 `json:"max_cpu_of_limit"`     // max % of limit (for classification)
	AvgRAMOfLimit      float64 `json:"avg_ram_of_limit"`     // avg % of limit (for classification)
	MaxRAMOfLimit      float64 `json:"max_ram_of_limit"`     // max % of limit (for classification)
	Classification     string  `json:"classification"`       // combined: "needs_more_resources", "overprovisioned", "balanced"
	CPUClassification  string  `json:"cpu_classification"`   // "high", "low", "ok" (based on LIMIT)
	RAMClassification  string  `json:"ram_classification"`    // "high", "low", "ok" (based on LIMIT)
}

// ClassifyResourceOfLimit classifies a single resource (CPU or RAM) as "high", "low", or "ok"
// based on LIMIT utilization thresholds.
// - "high" if avg_of_limit >= 50%
// - "low" if avg_of_limit <= 10%
// - "ok" otherwise
func ClassifyResourceOfLimit(avgOfLimit float64) string {
	if avgOfLimit >= 50 {
		return "high"
	}
	if avgOfLimit <= 10 {
		return "low"
	}
	return "ok"
}

// ClassifyDeploymentOfLimit returns the combined classification based on LIMIT-based CPU and RAM metrics.
func ClassifyDeploymentOfLimit(avgCPUOfLimit, avgRAMOfLimit float64) string {
	cpuClass := ClassifyResourceOfLimit(avgCPUOfLimit)
	ramClass := ClassifyResourceOfLimit(avgRAMOfLimit)

	if cpuClass == "high" || ramClass == "high" {
		return "needs_more_resources"
	}
	if cpuClass == "low" && ramClass == "low" {
		return "overprovisioned"
	}
	return "balanced"
}

// ClassifyResource classifies a single resource (CPU or RAM) as "high", "low", or "ok".
// Deprecated: Use ClassifyResourceOfLimit for LIMIT-based classification.
func ClassifyResource(avg, max, highThreshold, lowThreshold float64) string {
	if avg >= highThreshold || max > 100 {
		return "high"
	}
	if avg <= lowThreshold && max <= 100 {
		return "low"
	}
	return "ok"
}

// ClassifyDeployment returns the combined classification based on both CPU and RAM metrics.
// Deprecated: Use ClassifyDeploymentOfLimit for LIMIT-based classification.
func ClassifyDeployment(avgCPU, maxCPU, cpuHighThreshold, cpuLowThreshold, avgRAM, maxRAM, ramHighThreshold, ramLowThreshold float64) string {
	cpuClass := ClassifyResource(avgCPU, maxCPU, cpuHighThreshold, cpuLowThreshold)
	ramClass := ClassifyResource(avgRAM, maxRAM, ramHighThreshold, ramLowThreshold)

	if cpuClass == "high" || ramClass == "high" {
		return "needs_more_resources"
	}
	if cpuClass == "low" && ramClass == "low" {
		return "overprovisioned"
	}
	return "balanced"
}

// ContainerMetrics holds raw metrics for internal processing.
type ContainerMetrics struct {
	DeploymentName string    `json:"deployment_name"`
	Namespace      string    `json:"namespace"`
	CPUPercent     float64   `json:"cpu_percent"`
	RAMPercent     float64   `json:"ram_percent"`
	Timestamp      time.Time `json:"timestamp"`
}
