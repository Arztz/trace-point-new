package profiler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/iterator"
)

// Client is the GCP Cloud Profiler client.
type Client struct {
	projectID string
	enabled   bool
}

// ProfileResult contains the top function identified from profiling data.
type ProfileResult struct {
	FunctionName string  `json:"function_name"`
	FilePath     string  `json:"file_path"`
	CPUPercent   float64 `json:"cpu_percent"`
	SampleCount  int64   `json:"sample_count"`
	ProfileType  string  `json:"profile_type"`
}

// NewClient creates a new GCP Cloud Profiler client.
func NewClient(projectID string, enabled bool) *Client {
	return &Client{
		projectID: projectID,
		enabled:   enabled,
	}
}

// GetCulpritFunction fetches GCP Profiler data and extracts the top CPU-consuming function.
func (c *Client) GetCulpritFunction(ctx context.Context, serviceName string, start, end time.Time) (*ProfileResult, error) {
	if !c.enabled {
		log.Printf("[Profiler] GCP Profiler is disabled, skipping")
		return nil, nil
	}

	if c.projectID == "" {
		log.Printf("[Profiler] No GCP project ID configured, skipping")
		return nil, nil
	}
	log.Printf("[Profiler] start fetch")
	profiles, err := c.fetchProfiles(ctx, serviceName, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profiles: %w", err)
	}

	if len(profiles) == 0 {
		log.Printf("[Profiler] No profiles found for service %s between %s and %s", serviceName, start, end)
		return nil, nil
	}

	// Find the profile with highest CPU consumption
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].CPUPercent > profiles[j].CPUPercent
	})

	return &profiles[0], nil
}

// fetchProfiles retrieves profiling data from GCP Cloud Profiler API
// using the ExportClient which provides the ListProfiles method.
func (c *Client) fetchProfiles(ctx context.Context, serviceName string, start, end time.Time) ([]ProfileResult, error) {
	// ListProfiles is on ExportClient, not ProfilerClient
	client, err := cloudprofiler.NewExportClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create profiler export client: %w", err)
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s", c.projectID)

	req := &cloudprofilerpb.ListProfilesRequest{
		Parent:   parent,
		PageSize: 100,
	}

	var profiles []ProfileResult

	it := client.ListProfiles(ctx, req)
	for {
		profile, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating profiles: %w", err)
		}

		// Filter by time range
		if profile.Duration != nil && profile.StartTime != nil {
			profileStart := profile.StartTime.AsTime()
			profileEnd := profileStart.Add(profile.Duration.AsDuration())

			// GCP ListProfiles returns newer profiles first.
			// If this profile ended BEFORE our start window, then all subsequent profiles 
			// in the iterator will also be older, so we can stop fetching entirely!
			if profileEnd.Before(start) {
				break
			}

			// If it's newer than our window, skip it but keep looking
			if profileStart.After(end) {
				continue
			}
		}

		// Filter by deployment/service name
		if profile.Deployment != nil {
			target := profile.Deployment.Target
			if target != "" && !strings.Contains(strings.ToLower(target), strings.ToLower(serviceName)) {
				continue
			}
		}

		// Extract profile data
		result := ProfileResult{
			ProfileType: profile.ProfileType.String(),
		}

		// Use deployment information for function/file identification
		if profile.Deployment != nil {
			result.FunctionName = fmt.Sprintf("[%s] %s", profile.ProfileType.String(), profile.Deployment.Target)
			result.FilePath = profile.Deployment.Target
		}

		// Use profile labels for more detail if available
		if profile.Labels != nil {
			if fn, ok := profile.Labels["function"]; ok {
				result.FunctionName = fn
			}
			if fp, ok := profile.Labels["file"]; ok {
				result.FilePath = fp
			}
		}

		profiles = append(profiles, result)
	}

	log.Printf("[Profiler] Found %d profiles for service %s", len(profiles), serviceName)
	return profiles, nil
}

// IsEnabled returns whether the profiler integration is enabled.
func (c *Client) IsEnabled() bool {
	return c.enabled && c.projectID != ""
}
