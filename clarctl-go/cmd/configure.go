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
			huh.NewOption("Claude Sonnet 5.5 (Recommended Flagship)", "anthropic/claude-sonnet-5-5"),
			huh.NewOption("Claude Opus 5.5 (Deep Reasoning & Agentic Coding)", "anthropic/claude-opus-5-5"),
			huh.NewOption("Claude Haiku 5.5 (Ultra-Fast & Lightweight)", "anthropic/claude-haiku-5-5"),
			huh.NewOption("Claude Fable 5.1 (Long-Horizon Tasks)", "anthropic/claude-fable-5-1"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "openai":
		return []huh.Option[string]{
			huh.NewOption("GPT-6.1 Sol (Recommended High-Performance)", "openai/gpt-6.1-sol"),
			huh.NewOption("GPT-6 Astra (Frontier Flagship)", "openai/gpt-6-astra"),
			huh.NewOption("GPT-6 Luna (Fast & Cost-Optimized)", "openai/gpt-6-luna"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "groq":
		return []huh.Option[string]{
			huh.NewOption("OpenAI GPT-OSS 120B (Recommended Flagship)", "groq/openai/gpt-oss-120b"),
			huh.NewOption("OpenAI GPT-OSS 20B (Ultra-Fast ~1000 tps)", "groq/openai/gpt-oss-20b"),
			huh.NewOption("Llama 3.1 8B Instant (Lightweight)", "groq/llama-3.1-8b-instant"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "mistral":
		return []huh.Option[string]{
			huh.NewOption("Mistral Large 4 (Recommended Multimodal)", "mistral/mistral-large-latest"),
			huh.NewOption("Codestral (Code & SRE Specialist)", "mistral/codestral-latest"),
			huh.NewOption("Mistral Small 4 (Cost-Effective)", "mistral/mistral-small-latest"),
			huh.NewOption("Custom model name...", "custom"),
		}
	case "ollama":
		return []huh.Option[string]{
			huh.NewOption("DeepSeek-R1 (Local Reasoning)", "ollama/deepseek-r1"),
			huh.NewOption("Qwen2.5-Coder / Qwen3-Coder (Code Specialist)", "ollama/qwen2.5-coder"),
			huh.NewOption("Llama 3.3 (General Performance)", "ollama/llama3.3"),
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
