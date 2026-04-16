package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/integration/prometheus"
	"github.com/trace-point/trace-point-renew/internal/middleware"
	"github.com/trace-point/trace-point-renew/internal/repository"
	"github.com/trace-point/trace-point-renew/internal/service"
)

// DatasourceInstance holds the specific dependencies for a single datasource.
type DatasourceInstance struct {
	PromClient *prometheus.Client
	Analyzer   *service.Analyzer
	Correlator *service.Correlator
}

// Server holds the API server dependencies.
type Server struct {
	cfg        *config.Config
	router     chi.Router
	spikeRepo  *repository.SpikeRepo
	gravity    *service.GravityCalculator
	instances  map[string]*DatasourceInstance
}

// NewServer creates a new API server with all routes configured.
func NewServer(
	cfg *config.Config,
	spikeRepo *repository.SpikeRepo,
	gravity *service.GravityCalculator,
	instances map[string]*DatasourceInstance,
) *Server {
	s := &Server{
		cfg:       cfg,
		spikeRepo: spikeRepo,
		gravity:   gravity,
		instances: instances,
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(chimiddleware.Compress(5))

	// System endpoints
	r.Get("/health", s.handleHealth)
	r.Get("/metrics", s.handleMetrics)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/", s.handleAPIInfo)
		
		r.Get("/datasources", s.handleDatasources)

		// Spike events
		r.Get("/spikes", s.handleListSpikes)
		r.Get("/spikes/analyze", s.handleAnalyzeSpikes)
		r.Get("/spikes/{id}", s.handleGetSpike)
		r.Get("/spikes/{id}/details", s.handleGetSpikeDetails)
		r.Post("/spikes/{id}/retry", s.handleRetrySpikeCorrelation)

		// Timeline
		r.Get("/timeline", s.handleTimeline)

		// Export
		r.Get("/export", s.handleExport)
		r.Get("/export/refactoring", s.handleExportRefactoring)

		// Config & Scores
		r.Get("/config", s.handleConfig)
		r.Get("/gravity-scores", s.handleGravityScores)
	})

	// Serve frontend static files (production)
	fileServer := http.FileServer(http.Dir("web/dist"))
	r.Handle("/*", fileServer)

	s.router = r
	return s
}

// Router returns the chi router for HTTP serving.
func (s *Server) Router() chi.Router {
	return s.router
}

// getInstance returns the DatasourceInstance from the query param or the first available.
func (s *Server) getInstance(r *http.Request) (*DatasourceInstance, string, error) {
	dsName := r.URL.Query().Get("datasource")
	if dsName != "" {
		if ds, ok := s.instances[dsName]; ok {
			return ds, dsName, nil
		}
		return nil, "", fmt.Errorf("datasource %q not found", dsName)
	}

	// Default to first available if none specified
	for name, ds := range s.instances {
		return ds, name, nil
	}

	return nil, "", fmt.Errorf("no datasources configured")
}
