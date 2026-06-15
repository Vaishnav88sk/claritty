// Package config loads agent configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all agent runtime configuration.
type Config struct {
	ClusterName     string
	HubURL          string
	HubAPIKey       string
	LLMProvider     string
	LLMModel        string
	GroqAPIKey      string
	OpenAIAPIKey    string
	MistralAPIKey   string
	AnthropicAPIKey string
	OllamaHost      string
	ScanInterval    time.Duration
	Namespaces      []string // empty = all namespaces
	ListenAddr      string
}

// Load reads configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		ClusterName:     getEnv("CLARITTY_CLUSTER_NAME", "default-cluster"),
		HubURL:          getEnv("CLARITTY_HUB_URL", "http://localhost:8822"),
		HubAPIKey:       getEnv("CLARITTY_HUB_API_KEY", ""),
		LLMProvider:     getEnv("LLM_PROVIDER", "groq"),
		LLMModel:        getEnv("LLM_MODEL", "llama-3.3-70b-versatile"),
		GroqAPIKey:      getEnv("GROQ_API_KEY", ""),
		OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
		MistralAPIKey:   getEnv("MISTRAL_API_KEY", ""),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
		OllamaHost:      getEnv("OLLAMA_HOST", "http://localhost:11434"),
		ListenAddr:      getEnv("AGENT_LISTEN_ADDR", ":9090"),
	}

	secs, err := strconv.Atoi(getEnv("SCAN_INTERVAL_SECS", "300"))
	if err != nil || secs < 30 {
		secs = 300
	}
	cfg.ScanInterval = time.Duration(secs) * time.Second

	if ns := os.Getenv("WATCH_NAMESPACES"); ns != "" {
		for _, n := range strings.Split(ns, ",") {
			n = strings.TrimSpace(n)
			if n != "" {
				cfg.Namespaces = append(cfg.Namespaces, n)
			}
		}
	}
	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
