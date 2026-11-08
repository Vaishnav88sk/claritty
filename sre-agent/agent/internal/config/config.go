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

// DefaultModelForProvider returns the recommended default model for a given provider.
func DefaultModelForProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "anthropic":
		return "claude-sonnet-5-5"
	case "openai":
		return "gpt-6.1-sol"
	case "mistral":
		return "mistral-large-latest"
	case "ollama":
		return "llama3.3"
	case "groq":
		fallthrough
	default:
		return "openai/gpt-oss-120b"
	}
}

// Load reads configuration from environment variables.
func Load() *Config {
	provider := getEnv("LLM_PROVIDER", "groq")
	cfg := &Config{
		ClusterName:     getEnv("CLARITTY_CLUSTER_NAME", "default-cluster"),
		HubURL:          getEnv("CLARITTY_HUB_URL", "http://localhost:8822"),
		HubAPIKey:       getEnv("CLARITTY_HUB_API_KEY", ""),
		LLMProvider:     provider,
		LLMModel:        getEnv("LLM_MODEL", DefaultModelForProvider(provider)),
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
