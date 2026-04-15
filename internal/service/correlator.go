package service

import (
	"context"
	"log"
	"time"

	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/integration/profiler"
	"github.com/trace-point/trace-point-renew/internal/integration/signoz"
	"github.com/trace-point/trace-point-renew/internal/repository"
)

// Correlator orchestrates the full correlation chain:
// Deployment Spike → Route → Trace ID → Function Name
type Correlator struct {
	cfg            *config.Config
	signozClient   *signoz.Client
	profilerClient *profiler.Client
	spikeRepo      *repository.SpikeRepo
	alerter        *Alerter
}

// NewCorrelator creates a new correlator.
func NewCorrelator(
	cfg *config.Config,
	signozClient *signoz.Client,
	profilerClient *profiler.Client,
	spikeRepo *repository.SpikeRepo,
	alerter *Alerter,
) *Correlator {
	return &Correlator{
		cfg:            cfg,
		signozClient:   signozClient,
		profilerClient: profilerClient,
		spikeRepo:      spikeRepo,
		alerter:        alerter,
	}
}

// HandleSpike processes a detected spike through the full correlation pipeline.
// This runs asynchronously with a reconciliation buffer.
func (c *Correlator) HandleSpike(event *domain.SpikeEvent) {
	// Step 1: Store the initial spike event
	if err := c.spikeRepo.Create(event); err != nil {
		log.Printf("[Correlator] Failed to store spike event: %v", err)
		return
	}
	log.Printf("[Correlator] Spike stored: %s for %s/%s", event.ID, event.Namespace, event.DeploymentName)

	// Step 2: Wait for reconciliation buffer (let GCP Profiler data become available)
	bufferMinutes := c.cfg.Detection.ReconciliationBufferMinutes
	log.Printf("[Correlator] Waiting %d minutes for reconciliation buffer...", bufferMinutes)
	time.Sleep(time.Duration(bufferMinutes) * time.Minute)

	ctx := context.Background()

	// Step 3: Query SigNoz/ClickHouse for traces during the spike period
	log.Printf("[Correlator] Querying SigNoz for traces...")
	correlation, err := c.signozClient.CorrelateSpike(
		event.Namespace,
		event.DeploymentName,
		event.Timestamp,
		5, // ±5 minute window
	)
	if err != nil {
		log.Printf("[Correlator] SigNoz correlation failed (continuing without): %v", err)
	} else if correlation != nil && correlation.FoundTraces {
		event.RouteName = strPtr(correlation.CulpritRoute)
		event.TraceID = strPtr(correlation.TraceID)
		log.Printf("[Correlator] Found culprit route: %s (trace: %s)", correlation.CulpritRoute, correlation.TraceID)
	}

	// Step 4: Fetch GCP Profiler data
	if c.profilerClient.IsEnabled() {
		log.Printf("[Correlator] Fetching GCP Profiler data...")
		profileStart := event.Timestamp.Add(-10 * time.Minute)
		profileEnd := event.Timestamp.Add(5 * time.Minute)

		profileResult, err := c.profilerClient.GetCulpritFunction(ctx, event.DeploymentName, profileStart, profileEnd)
		if err != nil {
			log.Printf("[Correlator] GCP Profiler query failed (continuing without): %v", err)
		} else if profileResult != nil {
			event.CulpritFunction = strPtr(profileResult.FunctionName)
			event.CulpritFilePath = strPtr(profileResult.FilePath)
			log.Printf("[Correlator] Found culprit function: %s in %s", profileResult.FunctionName, profileResult.FilePath)
		}
	}

	// Step 5: Update spike event with correlation data
	if err := c.spikeRepo.Update(event); err != nil {
		log.Printf("[Correlator] Failed to update spike event: %v", err)
	}

	// Step 6: Send Discord alert
	if c.alerter != nil {
		if err := c.alerter.Alert(event); err != nil {
			log.Printf("[Correlator] Failed to send alert: %v", err)
		}
	}

	log.Printf("[Correlator] ✅ Correlation complete for %s/%s: Route=%s, Function=%s",
		event.Namespace, event.DeploymentName,
		ptrStr(event.RouteName), ptrStr(event.CulpritFunction))
}

// RetryCorrelation manually runs the correlation pipeline for an existing spike synchronously without delay.
func (c *Correlator) RetryCorrelation(ctx context.Context, spikeID string) (*domain.SpikeEvent, error) {
	event, err := c.spikeRepo.GetByID(spikeID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, nil // Not found
	}

	log.Printf("[Correlator] Manual retry started for spike: %s", spikeID)

	// Step 1: Query SigNoz
	correlation, err := c.signozClient.CorrelateSpike(
		event.Namespace,
		event.DeploymentName,
		event.Timestamp,
		5,
	)
	if err == nil && correlation != nil && correlation.FoundTraces {
		event.RouteName = strPtr(correlation.CulpritRoute)
		event.TraceID = strPtr(correlation.TraceID)
	}
	if err != nil {
		log.Printf("[Correlator] Manual retry started for spike:[Trace] err %v", err)
	}
	// Step 2: Fetch GCP Profiler
	if c.profilerClient.IsEnabled() {
		profileStart := event.Timestamp.Add(-10 * time.Minute)
		profileEnd := event.Timestamp.Add(5 * time.Minute)

		profileResult, err := c.profilerClient.GetCulpritFunction(ctx, event.DeploymentName, profileStart, profileEnd)
		if err == nil && profileResult != nil {
			event.CulpritFunction = strPtr(profileResult.FunctionName)
			event.CulpritFilePath = strPtr(profileResult.FilePath)
			log.Printf("[Correlator] Manual retry started for spike:[Profiler] found Profiler")
		}
		if err != nil {
			log.Printf("[Correlator] Manual retry started for spike:[Profiler] err %v", err)
		}
	}

	// Step 3: UpdateDB
	if err := c.spikeRepo.Update(event); err != nil {
		return nil, err
	}

	return event, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrStr(s *string) string {
	if s == nil {
		return "<unknown>"
	}
	return *s
}
