package domain

// GravityScore represents the Resource Gravity Score for a service/route.
// High Score = (High Resource Peak) × (Low Call Frequency)
type GravityScore struct {
	ServiceName          string  `json:"service_name"`
	RouteName            string  `json:"route_name"`
	SpikeCount           int     `json:"spike_count"`
	MaxCPUPercent        float64 `json:"max_cpu_percent"`
	MaxRAMPercent        float64 `json:"max_ram_percent"`
	AverageCPUPercent    float64 `json:"average_cpu_percent"`
	AverageRAMPercent    float64 `json:"average_ram_percent"`
	ResourceGravityScore float64 `json:"resource_gravity_score"`
	IsJobLike            bool    `json:"is_job_like"`
	Tags                 []string `json:"tags,omitempty"`
}

// GravityScoreResponse is the API response for gravity scores.
type GravityScoreResponse struct {
	Scores    []GravityScore `json:"scores"`
	Generated string         `json:"generated_at"`
	Period    string         `json:"period"` // e.g., "7 days"
}

// RefactoringRecommendation contains a recommendation for service separation.
type RefactoringRecommendation struct {
	ServiceName        string  `json:"service_name"`
	RouteName          string  `json:"route_name"`
	GravityScore       float64 `json:"gravity_score"`
	SuggestedAction    string  `json:"suggested_action"`
	Rationale          string  `json:"rationale"`
	SeparationStrategy string  `json:"separation_strategy"`
}

// RefactoringExport is the full JSON export for refactoring intelligence.
type RefactoringExport struct {
	GeneratedAt     string                      `json:"generated_at"`
	Period          string                      `json:"period"`
	TotalServices   int                         `json:"total_services"`
	HighPriority    int                         `json:"high_priority"`
	MediumPriority  int                         `json:"medium_priority"`
	Recommendations []RefactoringRecommendation `json:"recommendations"`
}

// ClassifyGravityImpact returns the impact level based on gravity score.
func ClassifyGravityImpact(score float64) string {
	switch {
	case score >= 6:
		return "high"
	case score >= 3:
		return "medium"
	default:
		return "low"
	}
}
