package profiler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/iterator"
)

// Client is the GCP Cloud Profiler client.
type Client struct {
	projectID     string
	envVersionTag string
	enabled       bool
}

// ProfileResult contains the top function identified from profiling data.
type ProfileResult struct {
	FunctionName string  `json:"function_name"`
	FilePath     string  `json:"file_path"`
	CPUPercent   float64 `json:"cpu_percent"`
	SampleCount  int64   `json:"sample_count"`
	ProfileType  string  `json:"profile_type"`
	Version      string  `json:"version"`
}

// NewClient creates a new GCP Cloud Profiler client.
func NewClient(projectID string, envVersionTag string, enabled bool) *Client {
	return &Client{
		projectID:     projectID,
		envVersionTag: envVersionTag,
		enabled:       enabled,
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

	// Filter for latest semver if envVersionTag is "v"
	if c.envVersionTag == "v" {
		var latest string
		for _, p := range profiles {
			if latest == "" || compareSemver(p.Version, latest) > 0 {
				latest = p.Version
			}
		}

		if latest != "" {
			var filtered []ProfileResult
			for _, p := range profiles {
				if p.Version == latest {
					filtered = append(filtered, p)
				}
			}
			profiles = filtered
		}
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
		PageSize: 1000,
	}

	var profiles []ProfileResult
	seenTargets := make(map[string]bool)

	it := client.ListProfiles(ctx, req)
	for {
		profile, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating profiles: %w", err)
		}
		var profileVersion string

		// Filter by deployment/service name FIRST since we know the target
		if profile.Deployment != nil {
			target := profile.Deployment.Target

			if !seenTargets[target] {
				log.Printf("[Profiler] Scanned Target (First occurrence): %v", target)
				seenTargets[target] = true
			}

			// Filter by environment/version tag if specified
			if profile.Deployment.Labels != nil {
				profileVersion = profile.Deployment.Labels["version"]
			}

			if c.envVersionTag != "" {
				if c.envVersionTag == "v" {
					if !strings.HasPrefix(profileVersion, "v") {
						continue
					}
				} else {
					if !strings.HasPrefix(profileVersion, c.envVersionTag+"-") && profileVersion != c.envVersionTag {
						continue
					}
				}
			}

			// log.Printf("[Profiler] Compare Target: %v, %v", strings.ToLower(target), strings.ToLower(serviceName))
			if target != "" && !strings.Contains(strings.ToLower(target), strings.ToLower(serviceName)) {
				continue
			}
		}

		// Filter by time range
		if profile.Duration != nil && profile.StartTime != nil {
			profileStart := profile.StartTime.AsTime()
			profileEnd := profileStart.Add(profile.Duration.AsDuration())

			// Check if profile overlaps with our time window
			if profileEnd.Before(start) || profileStart.After(end) {
				continue
			}
		}
		log.Printf("[Profiler] Profile: %v", profile)
		// Extract profile data
		result := ProfileResult{
			ProfileType: profile.ProfileType.String(),
			Version:     profileVersion,
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

func compareSemver(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(v1Parts) {
			n1, _ = strconv.Atoi(v1Parts[i])
		}
		if i < len(v2Parts) {
			n2, _ = strconv.Atoi(v2Parts[i])
		}
		if n1 < n2 {
			return -1
		} else if n1 > n2 {
			return 1
		}
	}
	return 0
}

// IsEnabled returns whether the profiler integration is enabled.
func (c *Client) IsEnabled() bool {
	return c.enabled && c.projectID != ""
}
