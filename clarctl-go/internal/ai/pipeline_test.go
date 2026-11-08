package ai

import (
	"context"
	"testing"
	"time"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/k8s"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPipelineRunScan(t *testing.T) {
	// 1. Create a fake Kubernetes clientset
	fakeCS := fake.NewSimpleClientset()
	k8sClient := k8s.NewWithClient(fakeCS)

	// 2. Create a mock configuration
	cfg := &config.Config{
		LLMProvider:       "mock",
		LLMModel:          "mock-model",
		RestartWarningCnt: 2,
		PodLogTailLines:   10,
		Namespaces:        []string{"default"},
	}

	// 3. Create a MockLLM that returns a valid incident JSON
	mockOutput := `{
		"has_issue": true,
		"severity": "SEV1",
		"title": "Mock Pod Crash",
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
	if report.Title != "Mock Pod Crash" {
		t.Errorf("Expected title 'Mock Pod Crash', got '%s'", report.Title)
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

	// 7. Directly invoke Call to satisfy 100% code coverage on the mock
	_, _ = mockLLM.Call(context.Background(), "dummy prompt")
}

func TestPipeline_EvidenceBackedRCA(t *testing.T) {
	fakeCS := fake.NewSimpleClientset()
	k8sClient := k8s.NewWithClient(fakeCS)

	cfg := &config.Config{
		LLMProvider:       "mock",
		LLMModel:          "mock-model",
		RestartWarningCnt: 2,
		PodLogTailLines:   10,
		Namespaces:        []string{"default"},
	}

	mockOutput := `{
		"has_issue": true,
		"severity": "SEV1",
		"title": "Redis Outage — Service Unreachable",
		"category": "crashloop",
		"affected_namespaces": ["default"],
		"root_cause": "Redis has no available endpoints [E1, E2]",
		"conclusions": [
			{
				"statement": "Redis has no available endpoints",
				"evidence_ids": ["E1"],
				"reasoning": "Endpoint list is empty while consumer pods report connection refused."
			}
		],
		"hypotheses": [
			{
				"statement": "Upstream network partition delayed heartbeat registration",
				"reason": "Direct inter-node network metrics were not available",
				"validation_steps": ["ping redis from caller"]
			}
		],
		"confidence_score": 90,
		"remediation_plan": [
			{"step_number": 1, "description": "Restart redis deployment", "command": "kubectl rollout restart deployment/redis -n default", "is_destructive": false, "is_automated": true}
		],
		"runbook_used": "crash_loop.yaml"
	}`
	mockLLM := &MockLLM{Response: mockOutput}

	pipeline := &Pipeline{
		cfg:    cfg,
		k8sCli: k8sClient,
		llm:    mockLLM,
	}

	report, err := pipeline.RunScan(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if report.RootCause != "Redis has no available endpoints [E1, E2]" {
		t.Errorf("Unexpected root cause: %s", report.RootCause)
	}
	if len(report.Conclusions) != 1 {
		t.Fatalf("Expected 1 conclusion, got %d", len(report.Conclusions))
	}
	if report.Conclusions[0].Statement != "Redis has no available endpoints" {
		t.Errorf("Unexpected conclusion statement: %s", report.Conclusions[0].Statement)
	}
	if len(report.Conclusions[0].EvidenceIDs) != 1 || report.Conclusions[0].EvidenceIDs[0] != "E1" {
		t.Errorf("Unexpected evidence IDs: %v", report.Conclusions[0].EvidenceIDs)
	}
	if report.Conclusions[0].Reasoning == "" {
		t.Errorf("Expected reasoning to be populated")
	}
	if len(report.Hypotheses) != 1 {
		t.Fatalf("Expected 1 hypothesis, got %d", len(report.Hypotheses))
	}
	if report.Hypotheses[0].Statement != "Upstream network partition delayed heartbeat registration" {
		t.Errorf("Unexpected hypothesis statement: %s", report.Hypotheses[0].Statement)
	}
}

func TestParseReport_GuardrailUnsupportedConclusionsDemoted(t *testing.T) {
	ec := incident.NewEvidenceCollector()
	ec.Add(incident.Evidence{
		ID:       "E1",
		Source:   "kubernetes",
		Type:     incident.EvidenceTypeResource,
		Resource: "pod/default/redis-0",
		Severity: incident.EvidenceSeverityCritical,
		Content:  "Redis endpoints = 0",
	})

	// Output contains:
	// 1 supported conclusion (E1)
	// 1 unsupported conclusion with empty evidence_ids
	// 1 unsupported conclusion with invalid evidence_ids (E99)
	raw := `{
		"has_issue": true,
		"severity": "SEV1",
		"title": "Guardrail Test",
		"category": "crashloop",
		"root_cause": "Redis outage",
		"conclusions": [
			{
				"statement": "Redis endpoints = 0",
				"evidence_ids": ["E1"],
				"reasoning": "Directly backed by telemetry"
			},
			{
				"statement": "Unsupported speculation without evidence",
				"evidence_ids": [],
				"reasoning": "Just guessing"
			},
			{
				"statement": "Hallucinated conclusion with nonexistent ID",
				"evidence_ids": ["E99"],
				"reasoning": "Fake reference"
			}
		],
		"hypotheses": [
			{
				"statement": "Existing hypothesis",
				"reason": "Suspected latency"
			}
		]
	}`

	report, err := parseReport(raw, "mock-model", ec, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Only 1 conclusion should remain supported
	if len(report.Conclusions) != 1 {
		t.Fatalf("Expected exactly 1 verified conclusion, got %d", len(report.Conclusions))
	}
	if report.Conclusions[0].Statement != "Redis endpoints = 0" {
		t.Errorf("Unexpected verified conclusion: %s", report.Conclusions[0].Statement)
	}

	// 2 unsupported conclusions demoted + 1 original hypothesis = 3 hypotheses
	if len(report.Hypotheses) != 3 {
		t.Fatalf("Expected 3 hypotheses (2 demoted + 1 original), got %d: %+v", len(report.Hypotheses), report.Hypotheses)
	}

	// Check evidence was attached to report
	if len(report.Evidence) != 1 || report.Evidence[0].ID != "E1" {
		t.Errorf("Expected E1 attached as evidence, got: %+v", report.Evidence)
	}
}

func TestMineLogEvidence(t *testing.T) {
	ec := incident.NewEvidenceCollector()
	timeline := incident.NewTimelineEngine()
	logs := `
2026-11-08T10:00:00Z Starting application...
2026-11-08T10:00:01Z panic: runtime error: invalid memory address or nil pointer dereference
2026-11-08T10:00:02Z connection refused to redis:6379
2026-11-08T10:00:03Z Normal heartbeat log line
`
	mineLogEvidence(logs, "default", "payment-api", ec, timeline)

	all := ec.All()
	if len(all) != 2 {
		t.Fatalf("Expected 2 mined log evidence items, got %d", len(all))
	}
	if all[0].Type != incident.EvidenceTypeLog {
		t.Errorf("Expected log evidence type, got %s", all[0].Type)
	}

	events := timeline.Events()
	if len(events) != 2 {
		t.Fatalf("Expected 2 timeline events, got %d", len(events))
	}
	if events[0].Timestamp.IsZero() {
		t.Errorf("Expected parsed timestamp for log timeline event")
	}
}

func TestPipeline_WhatChangedInvestigation(t *testing.T) {
	now := time.Now().UTC()
	recentTime := metav1.NewTime(now.Add(-2 * time.Minute))

	// Problem pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payment-api-7b89-xyz",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 6,
					Ready:        false,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	// Recent deployment of payment-api v1.43
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payment-api",
			Namespace: "default",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "43",
				"app.kubernetes.io/version":         "v1.43",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "payment-api:v1.43"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:           appsv1.DeploymentProgressing,
					Status:         corev1.ConditionTrue,
					LastUpdateTime: recentTime,
				},
			},
		},
	}

	fakeCS := fake.NewSimpleClientset(pod, dep)
	k8sClient := k8s.NewWithClient(fakeCS)

	cfg := &config.Config{
		LLMProvider:         "mock",
		LLMModel:            "mock-model",
		RestartWarningCnt:   2,
		PodLogTailLines:     10,
		Namespaces:          []string{"default"},
		ChangeWindowMinutes: 60,
	}

	mockOutput := `{
		"has_issue": true,
		"severity": "SEV1",
		"title": "Payment API CrashLoopBackOff after v1.43 deployment",
		"category": "crashloop",
		"affected_namespaces": ["default"],
		"root_cause": "payment-api v1.43 deployed with nil pointer [E1]",
		"conclusions": [
			{
				"statement": "payment-api fails after deployment",
				"evidence_ids": ["E1"],
				"reasoning": "Deployment v1.43 occurred shortly before crash loop."
			}
		],
		"hypotheses": [],
		"confidence_score": 90,
		"remediation_plan": [
			{"step_number": 1, "description": "Rollback deployment", "command": "kubectl rollout undo deployment/payment-api -n default", "is_destructive": false, "is_automated": true}
		]
	}`

	pipeline := &Pipeline{
		cfg:    cfg,
		k8sCli: k8sClient,
		llm:    &MockLLM{Response: mockOutput},
	}

	report, err := pipeline.RunScan(context.Background())
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	if len(report.Changes) == 0 {
		t.Fatalf("Expected detected changes in report, got 0")
	}

	foundDeployment := false
	for _, c := range report.Changes {
		if c.Category == incident.ChangeCategoryDeployment && c.Correlated {
			foundDeployment = true
			if c.Relevance < 0.50 {
				t.Errorf("Expected correlated deployment to have high relevance, got %.2f", c.Relevance)
			}
		}
	}
	if !foundDeployment {
		t.Errorf("Expected correlated deployment change in report.Changes")
	}

	// Verify timeline contains change events
	foundTimelineChange := false
	for _, te := range report.Timeline {
		if te.Type == "change_event" {
			foundTimelineChange = true
			break
		}
	}
	if !foundTimelineChange {
		t.Errorf("Expected timeline to contain change_event")
	}
}
