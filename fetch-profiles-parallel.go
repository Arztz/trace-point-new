package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/iterator"
)

const (
	numWorkers     = 50
	pageSize       = 1000
	maxInitialPage = 100 // Get this many page tokens to start workers
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = "fundii-production" // fallback for testing
	}
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	startTime := time.Now()

	fmt.Printf("Starting PARALLEL profile fetch from project: %s\n", projectID)
	fmt.Printf("Workers: %d, PageSize: %d\n", numWorkers, pageSize)
	fmt.Printf("Start time: %s\n\n", startTime.Format(time.RFC3339))
	log.Printf("[Main] Initializing GCP Profiler client...")

	var totalProfiles int64
	var totalErrors int64

	client, err := cloudprofiler.NewExportClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create profiler export client: %v", err)
	}
	defer client.Close()

	parent := fmt.Sprintf("projects/%s", projectID)

	// Thread-safe aggregators
	var mu sync.Mutex
	serviceTargets := make(map[string]int)
	versions := make(map[string]int)
	profileTimes := make([]time.Time, 0, 1000)

	// Channel for page tokens to fetch
	pageTokenChan := make(chan string, numWorkers*2)
	resultChan := make(chan *cloudprofilerpb.Profile, 1000)

	// Worker function
	worker := func(workerID int, tokens <-chan string, results chan<- *cloudprofilerpb.Profile) {
		client, err := cloudprofiler.NewExportClient(ctx)
		if err != nil {
			log.Printf("[Worker %d] Failed to create client: %v", workerID, err)
			return
		}
		defer client.Close()

		for token := range tokens {
			req := &cloudprofilerpb.ListProfilesRequest{
				Parent:    parent,
				PageSize:  pageSize,
				PageToken: token,
			}

			it := client.ListProfiles(ctx, req)
			count := 0
			for {
				profile, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					atomic.AddInt64(&totalErrors, 1)
					log.Printf("[Worker %d] Error: %v", workerID, err)
					break
				}
				count++
				results <- profile
			}
			if count > 0 {
				log.Printf("[Worker %d] Fetched %d profiles", workerID, count)
			}
		}
	}

	// Start workers
	log.Printf("Starting %d workers...", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i, pageTokenChan, resultChan)
	}

	// Result collector goroutine
	doneChan := make(chan struct{})
	go func() {
		for profile := range resultChan {
			atomic.AddInt64(&totalProfiles, 1)
			idx := atomic.AddInt64(&totalProfiles, 0)
			if idx%5000 == 0 {
				fmt.Printf("Processed %d profiles...\n", idx)
			}

			if profile.Deployment != nil {
				mu.Lock()
				target := profile.Deployment.Target
				serviceTargets[target]++
				if profile.Deployment.Labels != nil {
					if version := profile.Deployment.Labels["version"]; version != "" {
						versions[version]++
					}
				}
				if profile.StartTime != nil {
					profileTimes = append(profileTimes, profile.StartTime.AsTime())
				}
				mu.Unlock()
			}
		}
		doneChan <- struct{}{}
	}()

	// Fetch first page to get initial page tokens
	fmt.Println("\nFetching first page to discover page tokens...")
	log.Printf("[Main] Sending first ListProfiles request with PageSize=%d...", pageSize)
	firstReq := &cloudprofilerpb.ListProfilesRequest{
		Parent:   parent,
		PageSize: pageSize,
	}

	firstIt := client.ListProfiles(ctx, firstReq)
	profileCount := 0
	lastLog := time.Now()

	for {
		profile, err := firstIt.Next()
		if err == iterator.Done {
			log.Printf("[Main] Iteration complete, no more profiles")
			break
		}
		if err != nil {
			log.Printf("Error fetching initial page: %v", err)
			break
		}
		profileCount++

		// Log progress every second
		if time.Since(lastLog) > time.Second {
			log.Printf("[Main] Fetched %d profiles so far... (%.1f profiles/sec)",
				profileCount, float64(profileCount)/time.Since(startTime).Seconds())
			lastLog = time.Now()
		}

		resultChan <- profile

		// Extract page token from iterator metadata if available
		// Note: iterator.Next() doesn't expose the token directly
		// We need to rely on pagination via initial request
	}

	// The iterator doesn't expose NextPageToken directly, so we need
	// to track profiles seen and assume pagination is handled internally
	// For true parallel fetch, we'd need to use raw HTTP API

	// For now, let's try a different approach: fetch multiple pages
	// by using the same initial token approach but with goroutines

	fmt.Printf("Initial page fetched: %d profiles\n", profileCount)
	fmt.Println("Note: GCP Profiler iterator doesn't expose page tokens directly.")
	fmt.Println("Parallel fetching requires raw REST API with page tokens.")

	// Close result channel and wait for completion
	close(resultChan)
	<-doneChan

	elapsed := time.Since(startTime)

	// Analyze time ordering
	var minTime, maxTime time.Time
	if len(profileTimes) > 0 {
		minTime = profileTimes[0]
		maxTime = profileTimes[0]
		for _, t := range profileTimes {
			if t.Before(minTime) {
				minTime = t
			}
			if t.After(maxTime) {
				maxTime = t
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PROFILE FETCH SUMMARY (Parallel)")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total profiles processed: %d\n", totalProfiles)
	fmt.Printf("Total errors: %d\n", totalErrors)
	fmt.Printf("Unique service targets: %d\n", len(serviceTargets))
	fmt.Printf("Unique versions: %d\n", len(versions))
	fmt.Printf("Time elapsed: %v\n", elapsed.Round(time.Millisecond))
	if totalProfiles > 0 {
		fmt.Printf("Profiles per second: %.0f\n", float64(totalProfiles)/elapsed.Seconds())
	}
	if len(profileTimes) > 0 {
		fmt.Printf("Time range: %v to %v\n", minTime.Format(time.RFC3339), maxTime.Format(time.RFC3339))
		fmt.Printf("Time span: %v\n", maxTime.Sub(minTime))
	}
	fmt.Println(strings.Repeat("=", 60))

	// Show top 10 service targets
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

	// Sort descending
	for i := 0; i < len(topTargets); i++ {
		for j := i + 1; j < len(topTargets); j++ {
			if topTargets[j].count > topTargets[i].count {
				topTargets[i], topTargets[j] = topTargets[j], topTargets[i]
			}
		}
	}

	for i := 0; i < 10 && i < len(topTargets); i++ {
		fmt.Printf("  %-45s %d\n", topTargets[i].name, topTargets[i].count)
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
