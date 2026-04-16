package service

import (
	"log"
	"sync"
	"time"

	"github.com/trace-point/trace-point-renew/internal/config"
	"github.com/trace-point/trace-point-renew/internal/domain"
	"github.com/trace-point/trace-point-renew/internal/integration/discord"
	"github.com/trace-point/trace-point-renew/internal/repository"
)

// Alerter manages Discord alerting with cooldown support.
type Alerter struct {
	cfg           *config.Config
	discordClient *discord.Client
	spikeRepo     *repository.SpikeRepo
	cooldowns     map[string]time.Time // key: "namespace/deployment" → cooldown end
	mu            sync.RWMutex
}

// NewAlerter creates a new alerter.
func NewAlerter(cfg *config.Config, discordClient *discord.Client, spikeRepo *repository.SpikeRepo) *Alerter {
	return &Alerter{
		cfg:           cfg,
		discordClient: discordClient,
		spikeRepo:     spikeRepo,
		cooldowns:     make(map[string]time.Time),
	}
}

// Alert sends a spike alert to Discord, respecting cooldown periods.
func (a *Alerter) Alert(event *domain.SpikeEvent) error {
	if !a.discordClient.IsEnabled() {
		log.Printf("[Alerter] Discord disabled, skipping alert for %s/%s/%s", event.Datasource, event.Namespace, event.DeploymentName)
		return nil
	}

	key := event.Datasource + "/" + event.Namespace + "/" + event.DeploymentName

	// Check cooldown
	a.mu.RLock()
	cooldownEnd, hasCooldown := a.cooldowns[key]
	a.mu.RUnlock()

	if hasCooldown && time.Now().Before(cooldownEnd) {
		log.Printf("[Alerter] Cooldown active for %s until %s, skipping", key, cooldownEnd.Format(time.RFC3339))
		return nil
	}

	// Send alert
	if err := a.discordClient.SendSpikeAlert(event); err != nil {
		return err
	}

	// Set cooldown
	cooldownDuration := time.Duration(a.cfg.Detection.CooldownMinutes) * time.Minute
	newCooldownEnd := time.Now().Add(cooldownDuration)

	a.mu.Lock()
	a.cooldowns[key] = newCooldownEnd
	a.mu.Unlock()

	// Update event
	event.AlertSent = true
	event.CooldownEnd = &newCooldownEnd

	if err := a.spikeRepo.Update(event); err != nil {
		log.Printf("[Alerter] Failed to update spike event after alert: %v", err)
	}

	log.Printf("[Alerter] ✅ Alert sent for %s, cooldown until %s", key, newCooldownEnd.Format(time.RFC3339))
	return nil
}
