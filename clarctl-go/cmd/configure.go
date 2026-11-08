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

		var modelChoice string
		form3 := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose LLM Model").
					Options(getModelOptions(provider)...).
					Value(&modelChoice),
			),
		)

		if err := form3.Run(); err != nil {
			return fmt.Errorf("configuration cancelled: %w", err)
		}

		selectedModel := modelChoice
		if modelChoice == "custom" {
			var customInput string
			formCustom := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter custom model name").
						Description("Example: claude-3-7-sonnet-latest, gpt-4o-mini, etc.").
						Value(&customInput).
						Validate(func(s string) error {
							if strings.TrimSpace(s) == "" {
								return fmt.Errorf("model name cannot be empty")
							}
							return nil
						}),
				),
			)
			if err := formCustom.Run(); err != nil {
				return fmt.Errorf("configuration cancelled: %w", err)
			}
			selectedModel = strings.TrimSpace(customInput)
		}

		model, err := updateConfig(envPath, provider, apiKey, selectedModel)
		if err != nil {
			return err
		}

		ui.PrintConfigSuccess(provider, model, envPath)
		return nil
	},
}

// getModelOptions returns real-world, active models for each provider.
func getModelOptions(provider string) []huh.Option[string] {
	switch provider {
	case "anthropic":
		return []huh.Option[string]{
			huh.NewOption("Claude 3.7 Sonnet (Latest Flagship)", "anthropic/claude-3-7-sonnet-latest"),
			huh.NewOption("Claude 3.5 Haiku (Fast & Lightweight)", "anthropic/claude-3-5-haiku-latest"),
			huh.NewOption("Claude 3.5 Sonnet (Previous Standard)", "anthropic/claude-3-5-sonnet-latest"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "openai":
		return []huh.Option[string]{
			huh.NewOption("GPT-4o (Omni Flagship)", "openai/gpt-4o"),
			huh.NewOption("GPT-4o Mini (Fast & Cost-Effective)", "openai/gpt-4o-mini"),
			huh.NewOption("o3-mini (Reasoning Model)", "openai/o3-mini"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "groq":
		return []huh.Option[string]{
			huh.NewOption("Llama 3.3 70B Versatile (Recommended)", "groq/llama-3.3-70b-versatile"),
			huh.NewOption("DeepSeek R1 Distill 70B (Reasoning)", "groq/deepseek-r1-distill-llama-70b"),
			huh.NewOption("Llama 3.1 8B Instant (Ultra-Fast)", "groq/llama-3.1-8b-instant"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "mistral":
		return []huh.Option[string]{
			huh.NewOption("Mistral Large (Flagship Multimodal)", "mistral/mistral-large-latest"),
			huh.NewOption("Codestral (Code & SRE Specialist)", "mistral/codestral-latest"),
			huh.NewOption("Mistral Small (Cost-Effective)", "mistral/mistral-small-latest"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "ollama":
		return []huh.Option[string]{
			huh.NewOption("Llama 3.3 (High Performance)", "ollama/llama3.3"),
			huh.NewOption("DeepSeek-R1 (Local Reasoning)", "ollama/deepseek-r1"),
			huh.NewOption("Qwen2.5-Coder (Code Specialist)", "ollama/qwen2.5-coder"),
			huh.NewOption("Custom model name...", "custom"),
		}
	default:
		return []huh.Option[string]{
			huh.NewOption("Default Model", config.DefaultModelForProvider(provider)),
			huh.NewOption("Custom model name...", "custom"),
		}
	}
}

// updateConfig updates the .env file with the selected provider, generating default or custom models.
func updateConfig(envPath, provider, apiKey string, customModel ...string) (string, error) {
	var envKey string
	defaultModel := config.DefaultModelForProvider(provider)

	switch provider {
	case "groq":
		envKey = "GROQ_API_KEY"
	case "mistral":
		envKey = "MISTRAL_API_KEY"
	case "openai":
		envKey = "OPENAI_API_KEY"
	case "anthropic":
		envKey = "ANTHROPIC_API_KEY"
	case "ollama":
		envKey = "OLLAMA_HOST"
	}

	modelToUse := defaultModel
	if len(customModel) > 0 && strings.TrimSpace(customModel[0]) != "" {
		modelToUse = strings.TrimSpace(customModel[0])
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
	envMap["LLM_MODEL"] = modelToUse
	envMap[envKey] = apiKey

	if err := godotenv.Write(envMap, envPath); err != nil {
		return "", fmt.Errorf("failed to write configuration: %w", err)
	}

	return modelToUse, nil
}
