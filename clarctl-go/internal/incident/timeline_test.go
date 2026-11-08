package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTimelineEngine_MultiSourceAndChronologicalOrder(t *testing.T) {
	engine := NewTimelineEngine()

	baseTime := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)

	// Add events out of order from multiple sources
	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(7 * time.Minute),
		Source:    "ai-sre",
		Type:      "alert",
		Resource:  "cluster",
		Message:   "Incident detected",
		Severity:  "critical",
	})

	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(2 * time.Minute),
		Source:    "deployments",
		Type:      "deployment",
		Resource:  "deployment/payment-api",
		Message:   "Deployment v1.43",
		Severity:  "info",
	})

	engine.Add(TimelineEvent{
		Timestamp:  baseTime.Add(3 * time.Minute),
		Source:     "kubernetes",
		Type:       "event",
		Resource:   "endpoints/redis",
		Message:    "Redis endpoints → 0",
		Severity:   "critical",
		EvidenceID: "E1",
	})

	engine.Add(TimelineEvent{
		Timestamp:  baseTime.Add(4 * time.Minute),
		Source:     "logs",
		Type:       "log",
		Resource:   "pod/payment-api",
		Message:    "Connection errors ↑",
		Severity:   "critical",
		EvidenceID: "E2",
	})

	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(5 * time.Minute),
		Source:    "metrics",
		Type:      "metric",
		Resource:  "service/payment-api",
		Message:   "API latency ↑",
		Severity:  "warning",
	})

	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(6 * time.Minute),
		Source:    "metrics",
		Type:      "metric",
		Resource:  "service/payment-api",
		Message:   "Error rate ↑",
		Severity:  "critical",
	})

	sorted := engine.Events()
	if len(sorted) != 6 {
		t.Fatalf("Expected 6 events, got %d", len(sorted))
	}

	// Verify exact chronological sequence
	expectedOrder := []string{
		"Deployment v1.43",
		"Redis endpoints → 0",
		"Connection errors ↑",
		"API latency ↑",
		"Error rate ↑",
		"Incident detected",
	}

	for i, expectedMsg := range expectedOrder {
		if sorted[i].Message != expectedMsg {
			t.Errorf("Step %d: expected '%s', got '%s'", i, expectedMsg, sorted[i].Message)
		}
	}
}

func TestTimelineEngine_MissingTimestamps(t *testing.T) {
	engine := NewTimelineEngine()
	now := time.Now().UTC()

	// 1 event with timestamp
	engine.Add(TimelineEvent{
		Timestamp: now,
		Source:    "deployments",
		Message:   "Valid timestamp event",
	})

	// 2 events with zero/missing timestamp
	engine.Add(TimelineEvent{
		Timestamp: time.Time{},
		Source:    "logs",
		Message:   "Missing timestamp log event",
	})

	engine.Add(TimelineEvent{
		Timestamp: time.Time{},
		Source:    "git",
		Message:   "Missing timestamp git event",
	})

	events := engine.Events()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Valid timestamp must come first
	if events[0].Message != "Valid timestamp event" {
		t.Errorf("Expected valid timestamp event first, got: %s", events[0].Message)
	}

	// Missing timestamp events are placed at the end without panicking
	if !events[1].Timestamp.IsZero() || !events[2].Timestamp.IsZero() {
		t.Errorf("Expected remaining events to have zero timestamps")
	}

	// Formatted string representation handles missing timestamp gracefully
	str := events[1].String()
	if !strings.Contains(str, "??:??") {
		t.Errorf("Expected ??:?? in missing timestamp String(), got: %s", str)
	}
}

func TestTimelineEngine_QueryAndFilters(t *testing.T) {
	engine := NewTimelineEngine()
	t0 := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)

	engine.Add(TimelineEvent{
		Timestamp: t0,
		Source:    "k8s-events",
		Type:      "event",
		Resource:  "pod/auth",
		Message:   "BackOff",
		Severity:  "warning",
	})

	engine.Add(TimelineEvent{
		Timestamp: t0.Add(5 * time.Minute),
		Source:    "logs",
		Type:      "log",
		Resource:  "pod/auth",
		Message:   "panic in auth handler",
		Severity:  "critical",
	})

	engine.Add(TimelineEvent{
		Timestamp: t0.Add(10 * time.Minute),
		Source:    "metrics",
		Type:      "metric",
		Resource:  "service/frontend",
		Message:   "HTTP 500 spike",
		Severity:  "critical",
	})

	// Query by time window
	between := engine.Between(t0.Add(1*time.Minute), t0.Add(8*time.Minute))
	if len(between) != 1 || between[0].Message != "panic in auth handler" {
		t.Errorf("Between filter mismatch: %+v", between)
	}

	// Query by source
	k8sEvts := engine.BySource("k8s-events")
	if len(k8sEvts) != 1 || k8sEvts[0].Resource != "pod/auth" {
		t.Errorf("BySource filter mismatch: %+v", k8sEvts)
	}

	// Query by resource
	authEvts := engine.ByResource("auth")
	if len(authEvts) != 2 {
		t.Errorf("ByResource filter mismatch, expected 2, got %d", len(authEvts))
	}

	// Query by severity
	critEvts := engine.Query(TimelineFilter{MinSeverity: "critical"})
	if len(critEvts) != 2 {
		t.Errorf("MinSeverity filter mismatch, expected 2, got %d", len(critEvts))
	}
}

func TestTimelineEngine_FormatForAI(t *testing.T) {
	engine := NewTimelineEngine()
	t0 := time.Date(2026, 11, 8, 14, 2, 0, 0, time.UTC)

	engine.Add(TimelineEvent{
		Timestamp:  t0,
		Source:     "deployments",
		Resource:   "payment-api",
		Message:    "Deployed v1.43",
		EvidenceID: "E10",
	})

	engine.Add(TimelineEvent{
		Timestamp: t0.Add(1 * time.Minute),
		Source:    "kubernetes",
		Resource:  "redis",
		Message:   "Endpoints dropped to 0",
	})

	aiPrompt := engine.FormatForAI(10)
	if !strings.Contains(aiPrompt, "14:02:00 [deployments] payment-api: Deployed v1.43 [E10]") {
		t.Errorf("AI prompt format missing line: %s", aiPrompt)
	}
	if !strings.Contains(aiPrompt, "14:03:00 [kubernetes] redis: Endpoints dropped to 0") {
		t.Errorf("AI prompt format missing line: %s", aiPrompt)
	}
}

func TestTimelineEngine_IngestEvidenceAndReportSerialization(t *testing.T) {
	engine := NewTimelineEngine()
	now := time.Now().UTC().Truncate(time.Second)

	ev := Evidence{
		ID:        "E1",
		Source:    "kubernetes",
		Type:      EvidenceTypeResource,
		Timestamp: now,
		Resource:  "pod/default/crashloop-0",
		Severity:  EvidenceSeverityCritical,
		Content:   "Container terminated with OOMKilled",
	}

	ingested := engine.IngestEvidence(ev)
	if ingested.EvidenceID != "E1" || ingested.Message != "Container terminated with OOMKilled" {
		t.Errorf("IngestEvidence mismatch: %+v", ingested)
	}

	report := NewReport()
	report.Timeline = engine.Events()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(decoded.Timeline) != 1 || decoded.Timeline[0].EvidenceID != "E1" {
		t.Errorf("Report timeline serialization error: %+v", decoded.Timeline)
	}
}
