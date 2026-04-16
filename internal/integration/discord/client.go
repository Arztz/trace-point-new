package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/trace-point/trace-point-renew/internal/domain"
)

// Client is the Discord webhook client.
type Client struct {
	webhookURL string
	enabled    bool
	httpClient *http.Client
}

// NewClient creates a new Discord webhook client.
func NewClient(webhookURL string, enabled bool) *Client {
	return &Client{
		webhookURL: webhookURL,
		enabled:    enabled,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DiscordEmbed represents a Discord embed message.
type DiscordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color"`
	Fields      []EmbedField   `json:"fields"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Footer      *EmbedFooter   `json:"footer,omitempty"`
}

// EmbedField represents a field in a Discord embed.
type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// EmbedFooter represents the footer of a Discord embed.
type EmbedFooter struct {
	Text string `json:"text"`
}

// DiscordMessage is the payload sent to the Discord webhook.
type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds"`
}

// SendSpikeAlert sends a formatted spike alert to Discord.
func (c *Client) SendSpikeAlert(event *domain.SpikeEvent) error {
	if !c.enabled || c.webhookURL == "" {
		log.Printf("[Discord] Webhook disabled or URL not configured, skipping alert")
		return nil
	}

	// Determine severity color
	color := 0xFF0000 // Red for critical
	severity := "CRITICAL"
	cpuDelta := event.CPUUsagePercent - event.MovingAveragePercent
	if cpuDelta < 30 {
		color = 0xFFA500 // Orange for medium
		severity = "WARNING"
	}

	title := fmt.Sprintf("[%s - %s] Resource Spike Detected - %s", severity, event.Datasource, event.DeploymentName)

	fields := []EmbedField{
		{
			Name:   "🌐 Datasource",
			Value:  fmt.Sprintf("`%s`", event.Datasource),
			Inline: false,
		},
		{
			Name:   "🎯 Deployment",
			Value:  fmt.Sprintf("`%s`", event.DeploymentName),
			Inline: true,
		},
		{
			Name:   "📦 Namespace",
			Value:  fmt.Sprintf("`%s`", event.Namespace),
			Inline: true,
		},
		{
			Name:   "📊 CPU Impact",
			Value:  fmt.Sprintf("+%.1f%% (from %.1f%% to %.1f%%)", cpuDelta, event.MovingAveragePercent, event.CPUUsagePercent),
			Inline: false,
		},
		{
			Name:   "💾 RAM Impact",
			Value:  fmt.Sprintf("%.1f%%", event.RAMUsagePercent),
			Inline: true,
		},
	}

	if event.RouteName != nil && *event.RouteName != "" {
		fields = append(fields, EmbedField{
			Name:   "🛣️ Route",
			Value:  fmt.Sprintf("`%s`", *event.RouteName),
			Inline: false,
		})
	}

	if event.CulpritFunction != nil && *event.CulpritFunction != "" {
		culpritValue := fmt.Sprintf("`%s`", *event.CulpritFunction)
		if event.CulpritFilePath != nil && *event.CulpritFilePath != "" {
			culpritValue = fmt.Sprintf("`%s` in `%s`", *event.CulpritFunction, *event.CulpritFilePath)
		}
		fields = append(fields, EmbedField{
			Name:   "🔍 Culprit",
			Value:  culpritValue,
			Inline: false,
		})
	}

	if event.TraceID != nil && *event.TraceID != "" {
		fields = append(fields, EmbedField{
			Name:   "🔗 Trace ID",
			Value:  fmt.Sprintf("`%s`", *event.TraceID),
			Inline: false,
		})
	}

	embed := DiscordEmbed{
		Title:     title,
		Color:     color,
		Fields:    fields,
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Footer: &EmbedFooter{
			Text: "Trace-Point Resource Correlation Engine",
		},
	}

	msg := DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}

	return c.send(msg)
}

func (c *Client) send(msg DiscordMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	// Retry with exponential backoff
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewReader(payload))
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
				continue
			}
			return fmt.Errorf("failed to send Discord webhook: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limited, wait and retry
			retryAfter := 5 * time.Second
			log.Printf("[Discord] Rate limited, retrying after %s", retryAfter)
			time.Sleep(retryAfter)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("[Discord] Alert sent successfully")
			return nil
		}

		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	return fmt.Errorf("failed to send Discord webhook after %d retries", maxRetries)
}

// IsEnabled returns whether the Discord integration is enabled.
func (c *Client) IsEnabled() bool {
	return c.enabled && c.webhookURL != ""
}
