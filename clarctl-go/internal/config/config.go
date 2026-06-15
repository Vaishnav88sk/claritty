// Package config loads all configuration from environment and ~/.claritty/.env
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the AI-SRE engine.
type Config struct {
	// LLM
	LLMProvider    string
	LLMModel       string
	LLMTemperature float64
	LLMMaxTokens   int

	// API Keys
	GroqAPIKey      string
	MistralAPIKey   string
	OpenAIAPIKey    string
	AnthropicAPIKey string
	OllamaHost      string

	// Kubernetes
	Namespaces     []string
	ScanAllNS      bool
	KubeconfigPath string

	// Prometheus
	PrometheusURL     string
	PrometheusEnabled bool
	MetricsWindow     string

	// Loki
	LokiURL     string
	LokiEnabled bool

	// Alerting
	SlackWebhookURL string
	AlertWebhookURL string
	AlertSeverities []string

	// Thresholds
	CPUWarningPct      float64
	CPUCriticalPct     float64
	MemWarningPct      float64
	MemCriticalPct     float64
	RestartWarningCnt  int
	RestartCriticalCnt int
	ErrRateWarningPct  float64
	ErrRateCriticalPct float64

	// Scan
	ScanIntervalSeconds int
	PodLogTailLines     int
	MaxEventsPerScan    int
	AgentTimeoutSeconds int

	// Storage
	DBPath        string
	RunbooksDir   string
	LogPath       string
	DryRun        bool
	AutoRemediate bool
}

// ClarittyDir returns the ~/.claritty user-level config directory.
func ClarittyDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claritty")
}

// Load reads config from ~/.claritty/.env (preferred) then local .env,
// then falls back to environment variables and sensible defaults.
func Load() (*Config, error) {
	clarittyDir := ClarittyDir()
	if err := os.MkdirAll(clarittyDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create ~/.claritty: %w", err)
	}

	// Load .env files — order matters: global overrides local defaults
	_ = godotenv.Load(filepath.Join(clarittyDir, ".env"))
	_ = godotenv.Load() // local .env if present

	cfg := &Config{
		// LLM
		LLMProvider:    getenv("LLM_PROVIDER", "groq"),
		LLMModel:       getenv("LLM_MODEL", "groq/llama-3.3-70b-versatile"),
		LLMTemperature: getenvFloat("LLM_TEMPERATURE", 0.1),
		LLMMaxTokens:   getenvInt("LLM_MAX_TOKENS", 2048),

		// API Keys
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		MistralAPIKey:   os.Getenv("MISTRAL_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OllamaHost:      getenv("OLLAMA_HOST", "http://localhost:11434"),

		// Kubernetes
		Namespaces:     strings.Split(getenv("K8S_NAMESPACES", "default"), ","),
		ScanAllNS:      getenvBool("K8S_SCAN_ALL", false),
		KubeconfigPath: os.Getenv("KUBECONFIG"),

		// Prometheus
		PrometheusURL:     getenv("PROMETHEUS_URL", "http://localhost:9090"),
		PrometheusEnabled: getenvBool("PROMETHEUS_ENABLED", true),
		MetricsWindow:     getenv("METRICS_WINDOW", "5m"),

		// Loki
		LokiURL:     getenv("LOKI_URL", "http://localhost:3100"),
		LokiEnabled: getenvBool("LOKI_ENABLED", false),

		// Alerting
		SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		AlertWebhookURL: os.Getenv("ALERT_WEBHOOK_URL"),
		AlertSeverities: strings.Split(getenv("ALERT_SEVERITIES", "SEV1,SEV2"), ","),

		// Thresholds
		CPUWarningPct:      getenvFloat("CPU_WARNING_PCT", 80.0),
		CPUCriticalPct:     getenvFloat("CPU_CRITICAL_PCT", 95.0),
		MemWarningPct:      getenvFloat("MEMORY_WARNING_PCT", 85.0),
		MemCriticalPct:     getenvFloat("MEMORY_CRITICAL_PCT", 95.0),
		RestartWarningCnt:  getenvInt("RESTART_WARNING_COUNT", 5),
		RestartCriticalCnt: getenvInt("RESTART_CRITICAL_COUNT", 15),
		ErrRateWarningPct:  getenvFloat("ERROR_RATE_WARNING_PCT", 1.0),
		ErrRateCriticalPct: getenvFloat("ERROR_RATE_CRITICAL_PCT", 5.0),

		// Scan
		ScanIntervalSeconds: getenvInt("SCAN_INTERVAL_SECONDS", 300),
		PodLogTailLines:     getenvInt("POD_LOG_TAIL_LINES", 100),
		MaxEventsPerScan:    getenvInt("MAX_EVENTS_PER_SCAN", 50),
		AgentTimeoutSeconds: getenvInt("AGENT_TIMEOUT_SECONDS", 120),

		// Storage
		DBPath:        getenv("DB_PATH", filepath.Join(clarittyDir, "clarctl.db")),
		LogPath:       filepath.Join(clarittyDir, "clarctl.log"),
		RunbooksDir:   getenv("RUNBOOKS_DIR", ""),
		DryRun:        getenvBool("DRY_RUN", true),
		AutoRemediate: getenvBool("AUTO_REMEDIATE", false),
	}

	return cfg, nil
}

// Validate checks that required API keys are present for the chosen LLM provider.
func (c *Config) Validate() error {
	switch c.LLMProvider {
	case "groq":
		if c.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when LLM_PROVIDER=groq. Add it to ~/.claritty/.env")
		}
		os.Setenv("GROQ_API_KEY", c.GroqAPIKey)
	case "openai":
		if c.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when LLM_PROVIDER=openai")
		}
		os.Setenv("OPENAI_API_KEY", c.OpenAIAPIKey)
	case "mistral":
		if c.MistralAPIKey == "" {
			return fmt.Errorf("MISTRAL_API_KEY is required when LLM_PROVIDER=mistral")
		}
		os.Setenv("MISTRAL_API_KEY", c.MistralAPIKey)
	case "anthropic":
		if c.AnthropicAPIKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY is required when LLM_PROVIDER=anthropic")
		}
		os.Setenv("ANTHROPIC_API_KEY", c.AnthropicAPIKey)
	case "ollama":
		if c.OllamaHost == "" {
			return fmt.Errorf("OLLAMA_HOST is required when LLM_PROVIDER=ollama")
		}
		os.Setenv("OLLAMA_HOST", c.OllamaHost)
	}
	return nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return strings.ToLower(v) == "true"
	}
	return def
}
