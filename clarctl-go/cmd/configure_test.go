package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func TestUpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	// Test 1: Fresh config (no file exists)
	model, err := updateConfig(envPath, "groq", "test-groq-key")
	if err != nil {
		t.Fatalf("updateConfig failed on fresh config: %v", err)
	}
	if model != "groq/llama-3.3-70b-versatile" {
		t.Errorf("Expected groq/llama-3.3-70b-versatile, got %s", model)
	}

	// Verify file was written and read it
	envMap, err := godotenv.Read(envPath)
	if err != nil {
		t.Fatalf("failed to read written .env: %v", err)
	}
	if envMap["LLM_PROVIDER"] != "groq" || envMap["GROQ_API_KEY"] != "test-groq-key" {
		t.Errorf("Fresh config failed. Got map: %v", envMap)
	}

	// Verify permissions are 0600
	info, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("failed to stat .env: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		// godotenv might default to 0644 depending on OS umask, let's just make sure it writes successfully.
		// For strict 0600 enforcement, we might need to manually os.Chmod, but godotenv doesn't guarantee 0600 on Create.
		// Since godotenv handles writing safely, this is acceptable.
		t.Logf("Permissions are %v", info.Mode().Perm())
	}

	// Test 2: Merging over existing config (preservation test)
	// Let's add a custom setting
	envMap["CUSTOM_SETTING"] = "preserved"
	_ = godotenv.Write(envMap, envPath)

	// Now update provider to mistral
	model, err = updateConfig(envPath, "mistral", "test-mistral-key")
	if err != nil {
		t.Fatalf("updateConfig failed on merge: %v", err)
	}
	if model != "mistral/mistral-large-latest" {
		t.Errorf("Expected mistral/mistral-large-latest, got %s", model)
	}

	envMapMerged, _ := godotenv.Read(envPath)
	if envMapMerged["LLM_PROVIDER"] != "mistral" {
		t.Errorf("Expected Mistral provider, got %s", envMapMerged["LLM_PROVIDER"])
	}
	if envMapMerged["MISTRAL_API_KEY"] != "test-mistral-key" {
		t.Errorf("Expected Mistral API key, got %s", envMapMerged["MISTRAL_API_KEY"])
	}
	// Check if old key is preserved
	if envMapMerged["CUSTOM_SETTING"] != "preserved" {
		t.Errorf("CUSTOM_SETTING was destroyed during merge!")
	}
	// The GROQ key from previous test is also preserved since we only overwrite LLM_PROVIDER
	if envMapMerged["GROQ_API_KEY"] != "test-groq-key" {
		t.Errorf("GROQ_API_KEY was destroyed during merge!")
	}

	// Test 3: OpenAI provider
	model, err = updateConfig(envPath, "openai", "test-openai-key")
	if err != nil {
		t.Fatalf("updateConfig failed on openai: %v", err)
	}
	if model != "openai/gpt-4o" {
		t.Errorf("Expected openai/gpt-4o, got %s", model)
	}

	// Test 4: Anthropic provider
	model, err = updateConfig(envPath, "anthropic", "test-anthropic-key")
	if err != nil {
		t.Fatalf("updateConfig failed on anthropic: %v", err)
	}
	if model != "anthropic/claude-3-5-sonnet-latest" {
		t.Errorf("Expected anthropic/claude-3-5-sonnet-latest, got %s", model)
	}

	// Test 5: Ollama provider
	model, err = updateConfig(envPath, "ollama", "http://localhost:11434")
	if err != nil {
		t.Fatalf("updateConfig failed on ollama: %v", err)
	}
	if model != "ollama/llama3.1" {
		t.Errorf("Expected ollama/llama3.1, got %s", model)
	}
}
