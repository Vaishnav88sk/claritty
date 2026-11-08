package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAgentEvidenceModel_Serialization(t *testing.T) {
	val := 120.5
	now := time.Now().UTC().Truncate(time.Second)

	ev := Evidence{
		ID:        "E-101",
		Source:    "prometheus",
		Type:      EvidenceTypeMetric,
		Timestamp: now,
		Resource:  "redis",
		Severity:  EvidenceSeverityCritical,
		Content:   "Redis endpoints = 0",
		Metric:    "kube_endpoint_address_available",
		Value:     &val,
		Relevance: 0.95,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Evidence
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ID != "E-101" || decoded.Severity != EvidenceSeverityCritical {
		t.Errorf("Decoded mismatch: %+v", decoded)
	}
}

func TestAgentEvidenceCollector(t *testing.T) {
	ec := NewEvidenceCollector()

	ev1 := ec.Add(Evidence{
		Source:   "kubernetes",
		Type:     EvidenceTypeResource,
		Resource: "pod/default/redis-0",
		Severity: EvidenceSeverityCritical,
		Content:  "Redis endpoints = 0",
	})

	if ev1.ID != "E1" {
		t.Errorf("Expected auto-assigned ID 'E1', got '%s'", ev1.ID)
	}

	prompt := ec.FormatForPrompt()
	if !strings.Contains(prompt, "E1 [resource/kubernetes] (critical) pod/default/redis-0: Redis endpoints = 0") {
		t.Errorf("Prompt format mismatch: %s", prompt)
	}
}
