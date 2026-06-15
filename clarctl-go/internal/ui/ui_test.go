package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintBanner(t *testing.T) {
	// Temporarily redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set a custom version for the test
	originalVersion := Version
	Version = "v9.9.9-test"
	defer func() { Version = originalVersion }()

	// Call the function
	PrintBanner()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy from pipe: %v", err)
	}

	output := buf.String()

	// Verify the output contains the version string
	if !strings.Contains(output, "v9.9.9-test") {
		t.Errorf("Expected banner to contain dynamic version 'v9.9.9-test', got:\n%s", output)
	}

	// Verify the output contains Claritty branding
	if !strings.Contains(output, "AI-SRE Engine") {
		t.Errorf("Expected banner to contain 'AI-SRE Engine', got:\n%s", output)
	}
}

func TestPrintConfigSuccess(t *testing.T) {
	// Redirect stdout to capture the output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintConfigSuccess("test-provider", "test-model", "/test/path/.env")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Validate the rendered output contract
	if !strings.Contains(output, "Successfully configured Claritty!") {
		t.Errorf("Expected output to contain success header, got:\n%s", output)
	}
	if !strings.Contains(output, "Provider:") || !strings.Contains(output, "test-provider") {
		t.Errorf("Expected output to contain provider label and value, got:\n%s", output)
	}
	if !strings.Contains(output, "Model:") || !strings.Contains(output, "test-model") {
		t.Errorf("Expected output to contain model label and value, got:\n%s", output)
	}
	if !strings.Contains(output, "Path:") || !strings.Contains(output, "/test/path/.env") {
		t.Errorf("Expected output to contain path label and value, got:\n%s", output)
	}
}
