package domain

// RouteActivity represents the activity of a specific route during a spike period.
type RouteActivity struct {
	Route          string  `json:"route"`
	TraceCount     int     `json:"trace_count"`
	TotalDuration  int64   `json:"total_duration_ms"`
	ErrorCount     int     `json:"error_count"`
	ResourceWeight float64 `json:"resource_weight"`
}

// CorrelationResult contains the results of correlating a spike with trace data.
type CorrelationResult struct {
	FoundTraces     bool            `json:"found_traces"`
	TraceCount      int             `json:"trace_count"`
	RouteActivities []RouteActivity `json:"route_activities"`
	CulpritRoute    string          `json:"culprit_route"`
	CulpritScore    float64         `json:"culprit_score"`
	TraceID         string          `json:"trace_id"`
}

// ProfileResult contains the profiling information from GCP Cloud Profiler.
type ProfileResult struct {
	FunctionName string  `json:"function_name"`
	FilePath     string  `json:"file_path"`
	CPUPercent   float64 `json:"cpu_percent"`
	SampleCount  int64   `json:"sample_count"`
}

// FullCorrelation is the complete correlation chain:
// Deployment Spike → Route → Trace ID → Function Name
type FullCorrelation struct {
	SpikeID        string            `json:"spike_id"`
	DeploymentName string            `json:"deployment_name"`
	Namespace      string            `json:"namespace"`
	Correlation    *CorrelationResult `json:"correlation,omitempty"`
	Profiling      *ProfileResult     `json:"profiling,omitempty"`
}
