package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/integration/prometheus"
)

// MovingAverageEntry holds a single data point for moving average calculation.
type MovingAverageEntry struct {
	Timestamp  time.Time
	CPUPercent float64
	RAMPercent float64
}

// DeploymentState holds the in-memory state for spike detection per deployment.
type DeploymentState struct {
	History    []MovingAverageEntry
	LastSpike  time.Time
	InLearning bool
	StartedAt  time.Time
}

// Detector is the spike detection engine.
type Detector struct {
	cfg          *config.Config
	promClient   *prometheus.Client
	correlator   *Correlator
	states       map[string]*DeploymentState // key: "namespace/deployment"
	mu           sync.RWMutex
	onSpike      func(event *domain.SpikeEvent) // callback when spike detected
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewDetector creates a new spike detection engine.
func NewDetector(cfg *config.Config, promClient *prometheus.Client, correlator *Correlator) *Detector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Detector{
		cfg:        cfg,
		promClient: promClient,
		correlator: correlator,
		states:     make(map[string]*DeploymentState),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// SetOnSpike sets the callback function called when a spike is detected.
func (d *Detector) SetOnSpike(fn func(event *domain.SpikeEvent)) {
	d.onSpike = fn
}

// Start begins the spike detection polling loop.
func (d *Detector) Start() {
	interval := time.Duration(d.cfg.Detection.PollingIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[Detector] Starting spike detection (interval=%s, cpu_threshold=%.0f%%, ram_threshold=%.0f%%)",
		interval, d.cfg.Detection.CPUThreshold, d.cfg.Detection.MemoryThreshold)

	// Run immediately on start
	d.poll()

	for {
		select {
		case <-d.ctx.Done():
			log.Printf("[Detector] Stopped")
			return
		case <-ticker.C:
			d.poll()
		}
	}
}

// Stop stops the spike detection loop.
func (d *Detector) Stop() {
	d.cancel()
}

func (d *Detector) poll() {
	metrics, err := d.promClient.QueryInstantMetrics()
	if err != nil {
		log.Printf("[Detector] Failed to query metrics: %v", err)
		return
	}

	now := time.Now()

	for _, m := range metrics {
		if d.cfg.ShouldExcludeDeployment(m.DeploymentName) {
			continue
		}

		key := m.Namespace + "/" + m.DeploymentName
		state := d.getOrCreateState(key, now)

		// Add to history
		state.History = append(state.History, MovingAverageEntry{
			Timestamp:  now,
			CPUPercent: m.CPUPercent,
			RAMPercent: m.RAMPercent,
		})

		// Trim history to moving average window
		windowDuration := time.Duration(d.cfg.Detection.MovingAverageWindowMinutes) * time.Minute
		cutoff := now.Add(-windowDuration)
		trimmed := make([]MovingAverageEntry, 0)
		for _, h := range state.History {
			if h.Timestamp.After(cutoff) {
				trimmed = append(trimmed, h)
			}
		}
		state.History = trimmed

		// Check if still in learning period
		learningDuration := time.Duration(d.cfg.Detection.BaselineLearningPeriodMinutes) * time.Minute
		if now.Sub(state.StartedAt) < learningDuration {
			continue
		}
		state.InLearning = false

		// Calculate moving average
		if len(state.History) < 2 {
			continue
		}

		var sumCPU, sumRAM float64
		// Use all history except the current point
		for i := 0; i < len(state.History)-1; i++ {
			sumCPU += state.History[i].CPUPercent
			sumRAM += state.History[i].RAMPercent
		}
		count := float64(len(state.History) - 1)
		avgCPU := sumCPU / count
		avgRAM := sumRAM / count

		// Spike detection formula: Current > Moving Average + (Threshold %)
		// The threshold is applied as: MA * (1 + threshold/100)
		cpuThreshold := avgCPU * (1 + d.cfg.Detection.CPUThreshold/100)
		ramThreshold := avgRAM * (1 + d.cfg.Detection.MemoryThreshold/100)

		cpuSpike := m.CPUPercent > cpuThreshold && m.CPUPercent > 10 // Ignore very low absolute values
		ramSpike := m.RAMPercent > ramThreshold && m.RAMPercent > 10

		if !cpuSpike && !ramSpike {
			continue
		}

		// Check cooldown
		cooldownDuration := time.Duration(d.cfg.Detection.CooldownMinutes) * time.Minute
		if !state.LastSpike.IsZero() && now.Sub(state.LastSpike) < cooldownDuration {
			continue
		}

		// Spike detected!
		state.LastSpike = now
		movingAvg := avgCPU
		if ramSpike && !cpuSpike {
			movingAvg = avgRAM
		}

		event := &domain.SpikeEvent{
			Timestamp:            now,
			DeploymentName:       m.DeploymentName,
			Namespace:            m.Namespace,
			CPUUsagePercent:      m.CPUPercent,
			RAMUsagePercent:      m.RAMPercent,
			ThresholdPercent:     d.cfg.Detection.CPUThreshold,
			MovingAveragePercent: movingAvg,
		}

		log.Printf("[Detector] 🚨 SPIKE detected: %s/%s CPU=%.1f%% (avg=%.1f%%, threshold=%.1f%%) RAM=%.1f%% (avg=%.1f%%)",
			m.Namespace, m.DeploymentName, m.CPUPercent, avgCPU, cpuThreshold, m.RAMPercent, avgRAM)

		if d.onSpike != nil {
			go d.onSpike(event)
		}
	}
}

func (d *Detector) getOrCreateState(key string, now time.Time) *DeploymentState {
	d.mu.Lock()
	defer d.mu.Unlock()

	if state, ok := d.states[key]; ok {
		return state
	}

	state := &DeploymentState{
		History:    make([]MovingAverageEntry, 0),
		InLearning: true,
		StartedAt:  now,
	}
	d.states[key] = state
	return state
}

// GetState returns the current detection state for debugging/monitoring.
func (d *Detector) GetState() map[string]*DeploymentState {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cp := make(map[string]*DeploymentState, len(d.states))
	for k, v := range d.states {
		cp[k] = v
	}
	return cp
}
