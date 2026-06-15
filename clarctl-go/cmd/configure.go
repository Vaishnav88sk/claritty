package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Interactively configure Claritty CLI",
	Long:  "Launch an interactive wizard to configure your LLM provider and API keys.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var provider string

		form1 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your LLM Provider").
					Options(
						huh.NewOption("Groq (Recommended)", "groq"),
						huh.NewOption("Mistral", "mistral"),
						huh.NewOption("OpenAI", "openai"),
						huh.NewOption("Anthropic", "anthropic"),
						huh.NewOption("Ollama (Local)", "ollama"),
					).
					Value(&provider),
			),
		)

		if err := form1.Run(); err != nil {
			return fmt.Errorf("configuration cancelled: %w", err)
		}

		var apiKey string
		inputTitle := "Enter your API Key"
		inputDesc := "Your key will be securely saved and not displayed."
		echoMode := huh.EchoModePassword

		if provider == "ollama" {
			inputTitle = "Enter Ollama Host URL"
			inputDesc = "Example: http://localhost:11434"
			echoMode = huh.EchoModeNormal
			apiKey = "http://localhost:11434" // Default value
		}

		form2 := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(inputTitle).
					Description(inputDesc).
					EchoMode(echoMode).
					Value(&apiKey).
					Validate(func(s string) error {
						s = strings.TrimSpace(s)
						if s == "" {
							return fmt.Errorf("input cannot be empty")
						}
						if strings.Contains(s, "\n") || strings.Contains(s, "\r") {
							return fmt.Errorf("input cannot contain newlines")
						}
						return nil
					}),
			),
		)

		if err := form2.Run(); err != nil {
			return fmt.Errorf("configuration cancelled: %w", err)
		}

		apiKey = strings.TrimSpace(apiKey)
		envPath := filepath.Join(config.ClarittyDir(), ".env")

		model, err := updateConfig(envPath, provider, apiKey)
		if err != nil {
			return err
		}

		ui.PrintConfigSuccess(provider, model, envPath)
		return nil
	},
}

// updateConfig updates the .env file with the selected provider, generating default models.
func updateConfig(envPath, provider, apiKey string) (string, error) {
	var envKey string
	var defaultModel string

	switch provider {
	case "groq":
		envKey = "GROQ_API_KEY"
		defaultModel = "groq/llama-3.3-70b-versatile"
	case "mistral":
		envKey = "MISTRAL_API_KEY"
		defaultModel = "mistral/mistral-large-latest"
	case "openai":
		envKey = "OPENAI_API_KEY"
		defaultModel = "openai/gpt-4o"
	case "anthropic":
		envKey = "ANTHROPIC_API_KEY"
		defaultModel = "anthropic/claude-3-5-sonnet-latest"
	case "ollama":
		envKey = "OLLAMA_HOST"
		defaultModel = "ollama/llama3.1"
	}

	clarittyDir := filepath.Dir(envPath)
	if err := os.MkdirAll(clarittyDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	envMap, err := godotenv.Read(envPath)
	if err != nil {
		envMap = make(map[string]string)
	}

	envMap["LLM_PROVIDER"] = provider
	envMap["LLM_MODEL"] = defaultModel
	envMap[envKey] = apiKey

	if err := godotenv.Write(envMap, envPath); err != nil {
		return "", fmt.Errorf("failed to write configuration: %w", err)
	}

	return defaultModel, nil
}
