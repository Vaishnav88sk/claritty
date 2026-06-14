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
