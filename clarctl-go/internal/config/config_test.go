package config

import (
	"os"
	"testing"
)

func TestLoad_DefaultValues(t *testing.T) {
	// Ensure environment is clean
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLMProvider != "groq" {
		t.Errorf("expected default LLMProvider to be groq, got %s", cfg.LLMProvider)
	}
	if cfg.ScanIntervalSeconds != 300 {
		t.Errorf("expected default ScanIntervalSeconds to be 300, got %d", cfg.ScanIntervalSeconds)
	}
	if cfg.PrometheusEnabled != true {
		t.Errorf("expected default PrometheusEnabled to be true, got %v", cfg.PrometheusEnabled)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Clearenv()
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("SCAN_INTERVAL_SECONDS", "600")
	t.Setenv("PROMETHEUS_ENABLED", "false")
	t.Setenv("LLM_TEMPERATURE", "0.5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.LLMProvider != "openai" {
		t.Errorf("expected LLMProvider to be openai, got %s", cfg.LLMProvider)
	}
	if cfg.ScanIntervalSeconds != 600 {
		t.Errorf("expected ScanIntervalSeconds to be 600, got %d", cfg.ScanIntervalSeconds)
	}
	if cfg.PrometheusEnabled != false {
		t.Errorf("expected PrometheusEnabled to be false, got %v", cfg.PrometheusEnabled)
	}
	if cfg.LLMTemperature != 0.5 {
		t.Errorf("expected LLMTemperature to be 0.5, got %v", cfg.LLMTemperature)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "groq with key",
			cfg: &Config{
				LLMProvider: "groq",
				GroqAPIKey:  "test-key",
			},
			wantErr: false,
		},
		{
			name: "groq without key",
			cfg: &Config{
				LLMProvider: "groq",
			},
			wantErr: true,
		},
		{
			name: "openai with key",
			cfg: &Config{
				LLMProvider:  "openai",
				OpenAIAPIKey: "sk-test",
			},
			wantErr: false,
		},
		{
			name: "unknown provider",
			cfg: &Config{
				LLMProvider: "unknown",
			},
			wantErr: false, // Currently returns nil for unknown
		},
		{
			name: "anthropic with key",
			cfg: &Config{
				LLMProvider:     "anthropic",
				AnthropicAPIKey: "sk-ant",
			},
			wantErr: false,
		},
		{
			name: "anthropic without key",
			cfg: &Config{
				LLMProvider: "anthropic",
			},
			wantErr: true,
		},
		{
			name: "ollama with host",
			cfg: &Config{
				LLMProvider: "ollama",
				OllamaHost:  "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name: "ollama without host",
			cfg: &Config{
				LLMProvider: "ollama",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
