package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/charmbracelet/huh"
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
					Value(&apiKey),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("configuration cancelled: %w", err)
		}

		// Determine the environment variable name based on the provider
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

		clarittyDir := config.ClarittyDir()
		if err := os.MkdirAll(clarittyDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		envPath := filepath.Join(clarittyDir, ".env")
		content := fmt.Sprintf("LLM_PROVIDER=%s\nLLM_MODEL=%s\n%s=%s\n", provider, defaultModel, envKey, apiKey)

		if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write configuration: %w", err)
		}

		fmt.Printf("\n✨ Successfully configured Claritty!\n")
		fmt.Printf("Provider: %s\n", provider)
		fmt.Printf("Model:    %s\n", defaultModel)
		fmt.Printf("Config saved to: %s\n", envPath)

		return nil
	},
}
