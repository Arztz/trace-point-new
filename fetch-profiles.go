package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()

	projectID := "fundii-production"
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	startTime := time.Now()

	fmt.Printf("Starting profile fetch from project: %s\n", projectID)
	fmt.Printf("Start time: %s\n\n", startTime.Format(time.RFC3339))

	var totalProfiles int

	client, err := cloudprofiler.NewExportClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create profiler export client: %v", err)
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s", projectID)

	// Fetch first page to see page size behavior
	firstReq := &cloudprofilerpb.ListProfilesRequest{
		Parent:   parent,
		PageSize: 1000,
	}

	it := client.ListProfiles(ctx, firstReq)

	serviceTargets := make(map[string]int)
	versions := make(map[string]int)

	for {
		profile, err := it.Next()
		if err == iterator.Done {
			fmt.Println("\nNo more profiles to fetch")
			break
		}
		if err != nil {
			log.Printf("Error fetching profile: %v", err)
			break
		}

		totalProfiles++

		// Track unique service targets
		if profile.Deployment != nil {
			target := profile.Deployment.Target
			serviceTargets[target]++

			if profile.Deployment.Labels != nil {
				version := profile.Deployment.Labels["version"]
				if version != "" {
					versions[version]++
				}
			}
		}

		// Progress indicator every 1000 profiles
		if totalProfiles%1000 == 0 {
			fmt.Printf("Fetched %d profiles...\n", totalProfiles)
		}
	}

	elapsed := time.Since(startTime)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PROFILE FETCH SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total profiles fetched: %d\n", totalProfiles)
	fmt.Printf("Unique service targets: %d\n", len(serviceTargets))
	fmt.Printf("Unique versions: %d\n", len(versions))
	fmt.Printf("Time elapsed: %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Profiles per second: %.0f\n", float64(totalProfiles)/elapsed.Seconds())
	fmt.Println(strings.Repeat("=", 60))

	// Show top 10 service targets by profile count
	fmt.Println("\nTop 10 Service Targets:")
	fmt.Println(strings.Repeat("-", 40))

	type targetCount struct {
		name  string
		count int
	}
	var topTargets []targetCount
	for name, count := range serviceTargets {
		topTargets = append(topTargets, targetCount{name, count})
	}

	// Sort by count descending
	for i := 0; i < len(topTargets); i++ {
		for j := i + 1; j < len(topTargets); j++ {
			if topTargets[j].count > topTargets[i].count {
				topTargets[i], topTargets[j] = topTargets[j], topTargets[i]
			}
		}
	}

	for i := 0; i < 10 && i < len(topTargets); i++ {
		fmt.Printf("  %-40s %d\n", topTargets[i].name, topTargets[i].count)
	}

	// Show top versions
	fmt.Println("\nTop Versions:")
	fmt.Println(strings.Repeat("-", 40))

	var topVersions []targetCount
	for version, count := range versions {
		topVersions = append(topVersions, targetCount{version, count})
	}

	for i := 0; i < len(topVersions); i++ {
		for j := i + 1; j < len(topVersions); j++ {
			if topVersions[j].count > topVersions[i].count {
				topVersions[i], topVersions[j] = topVersions[j], topVersions[i]
			}
		}
	}

	for i := 0; i < 10 && i < len(topVersions); i++ {
		fmt.Printf("  %-20s %d\n", topVersions[i].name, topVersions[i].count)
	}

	fmt.Println()
}
