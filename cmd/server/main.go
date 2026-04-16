package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/trace-point/trace-point-renew/internal/api"
	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/integration/discord"
	"github.com/trace-point/trace-point-renew/internal/integration/profiler"
	"github.com/trace-point/trace-point-renew/internal/integration/prometheus"
	"github.com/trace-point/trace-point-renew/internal/integration/signoz"
	"github.com/trace-point/trace-point-renew/internal/repository"
	"github.com/trace-point/trace-point-renew/internal/service"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("🚀 Starting Trace-Point v1.0.3")

	// Load configuration
	configPath := "config.yaml"
	if envPath := os.Getenv("TRACE_POINT_CONFIG"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("✅ Configuration loaded from %s", configPath)

	// Initialize SQLite database
	db, err := repository.NewDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Printf("✅ Database initialized at %s", cfg.Database.Path)

	// Start auto-purge (7-day retention)
	db.StartPurgeLoop(7)

	// Create repositories
	spikeRepo := repository.NewSpikeRepo(db)

	// Global Discord client
	discordClient := discord.NewClient(
		cfg.Discord.WebhookURL,
		cfg.Discord.Enabled,
	)
	if discordClient.IsEnabled() {
		log.Printf("✅ Discord webhook enabled")
	} else {
		log.Printf("⚠️  Discord webhook disabled")
	}

	// Global services
	alerter := service.NewAlerter(cfg, discordClient, spikeRepo)
	gravity := service.NewGravityCalculator(spikeRepo)

	// Map of instances for API routing
	instances := make(map[string]*api.DatasourceInstance)
	var detectors []*service.Detector

	for i := range cfg.Datasources {
		ds := &cfg.Datasources[i]
		log.Printf("🚀 Initializing datasource: %s", ds.Name)

		promClient := prometheus.NewClient(
			ds.Prometheus.URL,
			ds.Prometheus.Timeout,
			ds.Namespaces,
		)

		signozClient := signoz.NewClient(
			ds.Signoz.URL,
			ds.Signoz.Database,
			ds.Signoz.User,
			ds.Signoz.Password,
			ds.Signoz.EnvTag,
			ds.Signoz.Timeout,
		)

		profilerClient := profiler.NewClient(
			ds.GCloud.ProjectID,
			ds.GCloud.EnvVersionTag,
			ds.GCloud.ProfilerEnabled,
		)

		correlator := service.NewCorrelator(cfg, signozClient, profilerClient, spikeRepo, alerter)
		analyzer := service.NewAnalyzer(cfg, promClient)

		detector := service.NewDetector(cfg, promClient, correlator, ds.Name)
		detector.SetOnSpike(func(event *domain.SpikeEvent) {
			correlator.HandleSpike(event)
		})

		instances[ds.Name] = &api.DatasourceInstance{
			PromClient: promClient,
			Analyzer:   analyzer,
			Correlator: correlator,
		}
		
		detectors = append(detectors, detector)

		// Start spike detection goroutine for this datasource
		go detector.Start()
		log.Printf("✅ Spike detector started for %s (interval=%ds)", ds.Name, cfg.Detection.PollingIntervalSeconds)
	}

	// Create API server
	server := api.NewServer(cfg, spikeRepo, gravity, instances)

	// Start HTTP server
	addr := cfg.GetListenAddr()
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh

		log.Printf("⏹️  Received signal %s, shutting down...", sig)

		for _, d := range detectors {
			d.Stop()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()

	log.Printf("🌐 Server listening on http://%s", addr)
	log.Printf("📊 Dashboard: http://localhost:%d", cfg.App.Port)
	log.Printf("🔌 API: http://localhost:%d/api/v1", cfg.App.Port)

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}

	log.Println("👋 Trace-Point stopped")
}
