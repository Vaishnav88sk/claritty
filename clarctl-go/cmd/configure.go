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
		var apiKey string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your LLM Provider").
					Options(
						huh.NewOption("Groq (Recommended)", "groq"),
						huh.NewOption("Mistral", "mistral"),
						huh.NewOption("OpenAI", "openai"),
					).
					Value(&provider),

				huh.NewInput().
					Title("Enter your API Key").
					Description("Your key will be securely saved and not displayed.").
					EchoMode(huh.EchoModePassword).
					Value(&apiKey).
					Validate(func(s string) error {
						s = strings.TrimSpace(s)
						if s == "" {
							return fmt.Errorf("API key cannot be empty")
						}
						if strings.Contains(s, "\n") || strings.Contains(s, "\r") {
							return fmt.Errorf("API key cannot contain newlines")
						}
						return nil
					}),
			),
		)

		if err := form.Run(); err != nil {
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
