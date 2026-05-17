// Package slack sends incident alerts to a Slack channel via webhooks.
package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client sends Slack messages via an Incoming Webhook.
type Client struct {
	webhookURL string
	channel    string
}

// New creates a Slack client. Returns nil if no webhook configured.
func New(webhookURL, channel string) *Client {
	if webhookURL == "" {
		return nil
	}
	return &Client{webhookURL: webhookURL, channel: channel}
}

// IncidentPayload is the data the hub passes for alerting.
type IncidentPayload struct {
	ID        string
	Cluster   string
	Severity  string
	Title     string
	Namespace string
	RootCause string
	HubURL    string
}

// AlertIncident sends a formatted Slack message for a new incident.
func (c *Client) AlertIncident(inc IncidentPayload) {
	if c == nil {
		return
	}

	sevEmoji := map[string]string{
		"SEV1": "🔴",
		"SEV2": "🟠",
		"SEV3": "🟡",
		"SEV4": "🟢",
	}
	emoji := sevEmoji[inc.Severity]
	if emoji == "" {
		emoji = "⚪"
	}

	text := fmt.Sprintf(
		"%s *[%s] %s*\n"+
			"*Cluster:* `%s` | *Namespace:* `%s`\n"+
			"*Root Cause:* %s\n"+
			"<%s/incidents/%s|👉 View in Dashboard>",
		emoji, inc.Severity, inc.Title,
		inc.Cluster, inc.Namespace,
		inc.RootCause,
		inc.HubURL, inc.ID,
	)

	payload := map[string]interface{}{
		"channel": c.channel,
		"text":    text,
	}
	body, _ := json.Marshal(payload)

	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Post(c.webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			fmt.Printf("Slack alert failed: %v\n", err)
			return
		}
		defer resp.Body.Close()
	}()
}
