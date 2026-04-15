package api

import (
	"net/http"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Return sanitized config (no secrets)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"app": map[string]interface{}{
			"host": s.cfg.App.Host,
			"port": s.cfg.App.Port,
		},
		"prometheus": map[string]interface{}{
			"url":                        s.cfg.Prometheus.URL,
			"timeout":                    s.cfg.Prometheus.Timeout.String(),
			"use_deployment_aggregation": s.cfg.Prometheus.UseDeploymentAggregation,
		},
		"detection": map[string]interface{}{
			"cpu_threshold":                    s.cfg.Detection.CPUThreshold,
			"memory_threshold":                 s.cfg.Detection.MemoryThreshold,
			"polling_interval_seconds":         s.cfg.Detection.PollingIntervalSeconds,
			"moving_average_window_minutes":    s.cfg.Detection.MovingAverageWindowMinutes,
			"baseline_learning_period_minutes": s.cfg.Detection.BaselineLearningPeriodMinutes,
			"reconciliation_buffer_minutes":    s.cfg.Detection.ReconciliationBufferMinutes,
			"cooldown_minutes":                 s.cfg.Detection.CooldownMinutes,
		},
		"timeline": map[string]interface{}{
			"cpu_close_to_100_threshold":  s.cfg.Timeline.CPUCloseTo100Threshold,
			"cpu_far_below_100_threshold": s.cfg.Timeline.CPUFarBelow100Threshold,
		},
		"database": map[string]interface{}{
			"type": s.cfg.Database.Type,
			"path": s.cfg.Database.Path,
		},
		"namespaces":              s.cfg.Namespaces,
		"deploy_exclude_patterns": s.cfg.DeployExcludePatterns,
		"discord": map[string]interface{}{
			"enabled": s.cfg.Discord.Enabled,
		},
		"gcloud": map[string]interface{}{
			"profiler_enabled": s.cfg.GCloud.ProfilerEnabled,
		},
	})
}
