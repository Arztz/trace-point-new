package main

import (
	"context"
	"fmt"
	"log"

	cloudprofiler "cloud.google.com/go/cloudprofiler/apiv2"
	"cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	"google.golang.org/api/iterator"
)

func main() {
	ctx := context.Background()
	client, err := cloudprofiler.NewExportClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	req := &cloudprofilerpb.ListProfilesRequest{
		Parent:   "projects/fundii-production",
		PageSize: 5,
	}

	fmt.Println("Connecting to GCP Cloud Profiler...")
	it := client.ListProfiles(ctx, req)

	count := 0
	for {
		profile, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error fetching profiles: %v", err)
		}
		count++
		fmt.Printf("✅ Found Profile %d: Type=%s, Target=%s, Duration=%s, StartTime=%s\n",
			count, profile.ProfileType.String(), profile.Deployment.Target, profile.Duration, profile.StartTime)

		if count >= 5 {
			break
		}
	}

	if count == 0 {
		fmt.Println("⚠️ Connected successfully, but no profiles exist in this project yet.")
	}
}
