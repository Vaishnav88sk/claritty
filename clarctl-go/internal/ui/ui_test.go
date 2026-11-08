package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
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
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	PrintConfigSuccess("test-provider", "test-model", "/test/path/.env")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
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

func TestPrintIncidentDetail_EvidenceBacked(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	report := incident.NewReport()
	report.ID = "INC-TEST1234"
	report.Title = "Redis Cluster Failure"
	report.Severity = incident.SEV1
	report.RootCause = "Redis has no available endpoints"

	report.Evidence = []incident.Evidence{
		{
			ID:       "E12",
			Source:   "kubernetes",
			Type:     incident.EvidenceTypeResource,
			Resource: "endpoints/default/redis",
			Severity: incident.EvidenceSeverityCritical,
			Content:  "Redis endpoints = 0",
		},
		{
			ID:       "E15",
			Source:   "logs",
			Type:     incident.EvidenceTypeLog,
			Resource: "pod/default/payment-api",
			Severity: incident.EvidenceSeverityCritical,
			Content:  "payment-api connection refused",
		},
	}

	report.Conclusions = []incident.Conclusion{
		{
			Statement:   "Redis has no available endpoints",
			EvidenceIDs: []string{"E12", "E15"},
			Reasoning:   "Endpoint list is empty while caller encounters connection refused.",
		},
	}

	report.Hypotheses = []incident.Hypothesis{
		{
			Statement:       "Potential kube-proxy routing issue",
			Reason:          "Packet loss between worker nodes",
			ValidationSteps: []string{"iptables-save check"},
		},
	}

	PrintIncidentDetail(report)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Redis has no available endpoints") {
		t.Errorf("Expected root cause in output, got:\n%s", output)
	}
	if !strings.Contains(output, "E12") || !strings.Contains(output, "Redis endpoints = 0") {
		t.Errorf("Expected E12 evidence in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Evidence-Backed Conclusions:") {
		t.Errorf("Expected conclusions header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Hypotheses (Unverified / Requires Validation):") {
		t.Errorf("Expected hypotheses header in output, got:\n%s", output)
	}
}
