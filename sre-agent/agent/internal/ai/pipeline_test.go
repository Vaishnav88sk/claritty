package ai

import (
	"context"
	"testing"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/k8s"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPipelineRunScan(t *testing.T) {
	// 1. Create a fake Kubernetes clientset
	fakeCS := fake.NewSimpleClientset()
	k8sClient := k8s.NewWithClient(fakeCS)

	// 2. Create a mock configuration
	cfg := &config.Config{
		ClusterName: "test-cluster",
		LLMProvider: "mock",
		LLMModel:    "mock-model",
		Namespaces:  []string{"default"},
	}

	// 3. Create a MockLLM that returns a valid incident JSON
	mockOutput := `{
		"has_issue": true,
		"severity": "SEV1",
		"title": "Mock Pod Crash in Agent",
		"category": "crashloop",
		"affected_namespaces": ["default"],
		"root_cause": "The pod is out of memory.",
		"confidence_score": 95,
		"remediation_plan": [
			{"step_number": 1, "description": "Delete pod", "command": "kubectl delete pod mock-pod -n default", "is_destructive": true, "is_automated": false}
		],
		"runbook_used": "crash_loop"
	}`
	mockLLM := &MockLLM{Response: mockOutput}

	// 4. Construct the pipeline manually with the injected mocks
	pipeline := &Pipeline{
		cfg:    cfg,
		k8sCli: k8sClient,
		llm:    mockLLM,
	}

	// 5. Run the scan
	report, err := pipeline.RunScan(context.Background())
	if err != nil {
		t.Fatalf("Expected no error from RunScan, got: %v", err)
	}

	// 6. Verify the report matches the mock LLM output
	if report.Title != "Mock Pod Crash in Agent" {
		t.Errorf("Expected title 'Mock Pod Crash in Agent', got '%s'", report.Title)
	}
	if !report.HasIssue {
		t.Errorf("Expected HasIssue to be true")
	}
	if report.ConfidenceScore != 95 {
		t.Errorf("Expected ConfidenceScore 95, got %d", report.ConfidenceScore)
	}
	if len(report.RemediationPlan) != 1 {
		t.Fatalf("Expected 1 remediation step, got %d", len(report.RemediationPlan))
	}
	if report.RemediationPlan[0].Command != "kubectl delete pod mock-pod -n default" {
		t.Errorf("Expected specific kubectl command, got '%s'", report.RemediationPlan[0].Command)
	}
}
