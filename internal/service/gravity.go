package service

import (
	"log"
	"regexp"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/repository"
)

// Job-like route patterns
var jobPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/tasks/`),
	regexp.MustCompile(`/batch/`),
	regexp.MustCompile(`/jobs/`),
	regexp.MustCompile(`/cron/`),
	regexp.MustCompile(`/workers/`),
}

// GravityCalculator computes Resource Gravity Scores.
type GravityCalculator struct {
	spikeRepo *repository.SpikeRepo
}

// NewGravityCalculator creates a new calculator.
func NewGravityCalculator(spikeRepo *repository.SpikeRepo) *GravityCalculator {
	return &GravityCalculator{spikeRepo: spikeRepo}
}

// CalculateScores computes Resource Gravity Scores for all services/routes.
// Formula: Score = Resource Peak × (1 / Request Frequency)
func (g *GravityCalculator) CalculateScores(datasource string, days int) ([]domain.GravityScore, error) {
	if days <= 0 {
		days = 7
	}

	events, err := g.spikeRepo.GetSpikesForGravity(datasource, days)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return []domain.GravityScore{}, nil
	}

	// Group by service+route
	type routeStats struct {
		service    string
		route      string
		maxCPU     float64
		maxRAM     float64
		totalCPU   float64
		totalRAM   float64
		count      int
		timestamps []time.Time
	}

	statsMap := make(map[string]*routeStats)

	for _, e := range events {
		route := "<unknown>"
		if e.RouteName != nil && *e.RouteName != "" {
			route = *e.RouteName
		}

		key := e.Namespace + "/" + e.DeploymentName + "/" + route
		if _, ok := statsMap[key]; !ok {
			statsMap[key] = &routeStats{
				service: e.Namespace + "/" + e.DeploymentName,
				route:   route,
			}
		}

		s := statsMap[key]
		s.count++
		s.totalCPU += e.CPUUsagePercent
		s.totalRAM += e.RAMUsagePercent
		if e.CPUUsagePercent > s.maxCPU {
			s.maxCPU = e.CPUUsagePercent
		}
		if e.RAMUsagePercent > s.maxRAM {
			s.maxRAM = e.RAMUsagePercent
		}
		s.timestamps = append(s.timestamps, e.Timestamp)
	}

	// Calculate gravity scores
	var scores []domain.GravityScore
	for _, s := range statsMap {
		// Resource Peak = max(CPU peak, RAM peak)
		resourcePeak := s.maxCPU
		if s.maxRAM > resourcePeak {
			resourcePeak = s.maxRAM
		}

		// Request Frequency = spikes per day
		var frequency float64
		if len(s.timestamps) >= 2 {
			timeSpan := s.timestamps[0].Sub(s.timestamps[len(s.timestamps)-1]).Hours() / 24
			if timeSpan > 0 {
				frequency = float64(s.count) / timeSpan
			}
		}
		if frequency == 0 {
			frequency = float64(s.count) / float64(days)
		}

		// Gravity Score = Resource Peak × (1 / Request Frequency)
		// Normalize peak to 0-10 scale
		normalizedPeak := resourcePeak / 100.0 * 10.0
		gravityScore := normalizedPeak
		if frequency > 0 {
			gravityScore = normalizedPeak * (1.0 / frequency)
		}

		// Check if route matches job-like patterns
		isJobLike := false
		var tags []string
		for _, pattern := range jobPatterns {
			if pattern.MatchString(s.route) {
				isJobLike = true
				tags = append(tags, "[Suspected Job]")
				break
			}
		}

		// Auto-tag based on behavior: high resource + low frequency
		if resourcePeak > 80 && frequency < 1 {
			if !isJobLike {
				tags = append(tags, "[Suspected Job]")
				isJobLike = true
			}
		}

		scores = append(scores, domain.GravityScore{
			ServiceName:          s.service,
			RouteName:            s.route,
			SpikeCount:           s.count,
			MaxCPUPercent:        s.maxCPU,
			MaxRAMPercent:        s.maxRAM,
			AverageCPUPercent:    s.totalCPU / float64(s.count),
			AverageRAMPercent:    s.totalRAM / float64(s.count),
			ResourceGravityScore: gravityScore,
			IsJobLike:            isJobLike,
			Tags:                 tags,
		})
	}

	// Sort by gravity score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].ResourceGravityScore > scores[i].ResourceGravityScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	log.Printf("[Gravity] Calculated %d gravity scores from %d spike events", len(scores), len(events))
	return scores, nil
}

// GenerateRefactoringReport generates a refactoring export with recommendations.
func (g *GravityCalculator) GenerateRefactoringReport(datasource string, days int) (*domain.RefactoringExport, error) {
	scores, err := g.CalculateScores(datasource, days)
	if err != nil {
		return nil, err
	}

	var recommendations []domain.RefactoringRecommendation
	var highPriority, mediumPriority int

	for _, s := range scores {
		impact := domain.ClassifyGravityImpact(s.ResourceGravityScore)

		var action, rationale, strategy string
		switch impact {
		case "high":
			highPriority++
			action = "Priority refactoring required"
			rationale = "High resource consumption with low request frequency indicates this route should be separated into a dedicated service"
			strategy = "Extract to dedicated microservice with independent scaling"
		case "medium":
			mediumPriority++
			action = "Consider optimization"
			rationale = "Moderate resource impact; evaluate if code optimization or caching can reduce footprint"
			strategy = "Optimize in-place or extract if complexity warrants"
		default:
			action = "Monitor"
			rationale = "Low impact; continue monitoring for trends"
			strategy = "No action needed"
		}

		if s.IsJobLike {
			action = "Extract as background job"
			strategy = "Move to async worker/queue with dedicated resource allocation"
		}

		recommendations = append(recommendations, domain.RefactoringRecommendation{
			ServiceName:        s.ServiceName,
			RouteName:          s.RouteName,
			GravityScore:       s.ResourceGravityScore,
			SuggestedAction:    action,
			Rationale:          rationale,
			SeparationStrategy: strategy,
		})
	}

	return &domain.RefactoringExport{
		GeneratedAt:     time.Now().Format(time.RFC3339),
		Period:          "7 days",
		TotalServices:   len(scores),
		HighPriority:    highPriority,
		MediumPriority:  mediumPriority,
		Recommendations: recommendations,
	}, nil
}
