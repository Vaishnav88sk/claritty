package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEvidenceModel_Serialization(t *testing.T) {
	val := 2.8
	now := time.Now().UTC().Truncate(time.Second)

	ev := Evidence{
		ID:        "E-123",
		Source:    "prometheus",
		Type:      EvidenceTypeMetric,
		Timestamp: now,
		Resource:  "payment-api",
		Severity:  EvidenceSeverityWarning,
		Content:   "High HTTP request latency",
		Metric:    "http_request_duration",
		Value:     &val,
		Relevance: 0.94,
		Metadata: map[string]interface{}{
			"threshold": 2.0,
			"cluster":   "prod-us-east",
		},
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Failed to marshal evidence: %v", err)
	}

	var decoded Evidence
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal evidence: %v", err)
	}

	if decoded.ID != "E-123" {
		t.Errorf("Expected ID 'E-123', got '%s'", decoded.ID)
	}
	if decoded.Type != EvidenceTypeMetric {
		t.Errorf("Expected type '%s', got '%s'", EvidenceTypeMetric, decoded.Type)
	}
	if decoded.Value == nil || *decoded.Value != 2.8 {
		t.Errorf("Expected metric value 2.8, got %v", decoded.Value)
	}
	if decoded.Relevance != 0.94 {
		t.Errorf("Expected relevance 0.94, got %f", decoded.Relevance)
	}
}

func TestEvidenceCollector(t *testing.T) {
	ec := NewEvidenceCollector()

	ev1 := ec.Add(Evidence{
		Source:   "kubernetes",
		Type:     EvidenceTypeResource,
		Resource: "pod/default/redis-0",
		Severity: EvidenceSeverityCritical,
		Content:  "Redis endpoints = 0",
	})

	ev2 := ec.Add(Evidence{
		Source:   "logs",
		Type:     EvidenceTypeLog,
		Resource: "pod/default/payment-api-abc",
		Severity: EvidenceSeverityCritical,
		Content:  "payment-api connection refused to redis:6379",
	})

	if ev1.ID != "E1" {
		t.Errorf("Expected auto-assigned ID 'E1', got '%s'", ev1.ID)
	}
	if ev2.ID != "E2" {
		t.Errorf("Expected auto-assigned ID 'E2', got '%s'", ev2.ID)
	}

	all := ec.All()
	if len(all) != 2 {
		t.Fatalf("Expected 2 evidence items, got %d", len(all))
	}

	found, ok := ec.GetByID("E1")
	if !ok || found.Content != "Redis endpoints = 0" {
		t.Errorf("Failed to retrieve E1 by ID")
	}

	promptStr := ec.FormatForPrompt()
	if !strings.Contains(promptStr, "E1 [resource/kubernetes] (critical) pod/default/redis-0: Redis endpoints = 0") {
		t.Errorf("Prompt format missing expected E1 line, got:\n%s", promptStr)
	}
	if !strings.Contains(promptStr, "E2 [log/logs] (critical) pod/default/payment-api-abc") {
		t.Errorf("Prompt format missing expected E2 line, got:\n%s", promptStr)
	}

	jsonStr, err := ec.ToJSON()
	if err != nil || !strings.Contains(jsonStr, "Redis endpoints = 0") {
		t.Errorf("Failed to serialize collector to JSON: %v, %s", err, jsonStr)
	}
}

func TestEvidenceStringRepresentation(t *testing.T) {
	val := 15.0
	ev := Evidence{
		ID:       "E12",
		Resource: "pod/default/crashloop-demo",
		Content:  "CrashLoopBackOff detected",
		Metric:   "restarts",
		Value:    &val,
	}

	str := ev.String()
	expected := "E12 - [pod/default/crashloop-demo] - CrashLoopBackOff detected - (metric: restarts=15.00)"
	if str != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, str)
	}
}

func TestReport_WithEvidenceBackedRCA(t *testing.T) {
	r := NewReport()
	r.Title = "Redis Outage"
	r.Severity = SEV1
	r.RootCause = "Redis has no available endpoints"

	r.Evidence = []Evidence{
		{
			ID:       "E12",
			Source:   "kubernetes",
			Type:     EvidenceTypeResource,
			Resource: "endpoints/default/redis",
			Severity: EvidenceSeverityCritical,
			Content:  "Redis endpoints = 0",
		},
		{
			ID:       "E15",
			Source:   "logs",
			Type:     EvidenceTypeLog,
			Resource: "pod/default/payment-api",
			Severity: EvidenceSeverityCritical,
			Content:  "payment-api connection refused",
		},
	}

	r.Conclusions = []Conclusion{
		{
			Statement:   "Redis has no available endpoints",
			EvidenceIDs: []string{"E12", "E15"},
			Reasoning:   "Zero endpoints in the service definition prevents upstream clients from establishing socket connections.",
		},
	}

	r.Hypotheses = []Hypothesis{
		{
			Statement:       "Potential kube-proxy routing failure on worker-2",
			Reason:          "Packet loss detected between worker nodes without direct endpoint failure evidence",
			ValidationSteps: []string{"Run iptables-save | grep redis", "Check kube-proxy logs on worker-2"},
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Failed to marshal Report: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Report: %v", err)
	}

	if len(decoded.Evidence) != 2 {
		t.Errorf("Expected 2 evidence items, got %d", len(decoded.Evidence))
	}
	if len(decoded.Conclusions) != 1 {
		t.Errorf("Expected 1 conclusion, got %d", len(decoded.Conclusions))
	}
	if len(decoded.Hypotheses) != 1 {
		t.Errorf("Expected 1 hypothesis, got %d", len(decoded.Hypotheses))
	}
	if decoded.Conclusions[0].EvidenceIDs[0] != "E12" {
		t.Errorf("Expected first evidence ID 'E12', got '%s'", decoded.Conclusions[0].EvidenceIDs[0])
	}
}
