// Package config loads hub configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	SlackWebhook  string
	SlackChannel  string
	HubAPIKey     string // optional shared secret for agent auth
	RequireAPIKey bool
}

func Load() *Config {
	requireAPIKey, _ := strconv.ParseBool(os.Getenv("CLARITTY_REQUIRE_HUB_API_KEY"))
	return &Config{
		Port:          getEnv("HUB_PORT", "8822"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		SlackWebhook:  getEnv("SLACK_WEBHOOK_URL", ""),
		SlackChannel:  getEnv("SLACK_CHANNEL", "#sre-alerts"),
		HubAPIKey:     getEnv("CLARITTY_HUB_API_KEY", ""),
		RequireAPIKey: requireAPIKey,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
