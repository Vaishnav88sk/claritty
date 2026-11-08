package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestChangeEngine_TimeWindowFiltering(t *testing.T) {
	refTime, _ := time.Parse("15:04", "14:05")
	window := 30 * time.Minute // 13:35 to 14:05

	engine := NewChangeEngine(window, refTime)

	// Change at 14:01 (inside window)
	tInside, _ := time.Parse("15:04", "14:01")
	accepted1 := engine.AddChange(ChangeRecord{
		Timestamp: tInside,
		Resource:  "deployment/default/payment-api",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})
	if !accepted1 {
		t.Errorf("Expected change at 14:01 to be accepted within 30m window")
	}

	// Change at 13:10 (outside window: >30m before 14:05)
	tOutside, _ := time.Parse("15:04", "13:10")
	accepted2 := engine.AddChange(ChangeRecord{
		Timestamp: tOutside,
		Resource:  "deployment/default/auth-service",
		Category:  ChangeCategoryDeployment,
		Summary:   "auth-service deployed",
	})
	if accepted2 {
		t.Errorf("Expected change at 13:10 to be rejected as outside 30m window")
	}

	if len(engine.All()) != 1 {
		t.Fatalf("Expected 1 accepted change, got %d", len(engine.All()))
	}
}

func TestChangeEngine_CorrelationAndMostRelevant(t *testing.T) {
	// Recreate exact example from Issue #51:
	// Incident started: 14:05
	// Potential changes:
	// 14:01 payment-api v1.43 deployed
	// 14:02 environment variable changed
	// 14:02 Redis Service modified
	// Most relevant: payment-api v1.43

	refTime, _ := time.Parse("15:04", "14:05")
	engine := NewChangeEngine(60*time.Minute, refTime)

	t1, _ := time.Parse("15:04", "14:01")
	t2, _ := time.Parse("15:04", "14:02")

	engine.AddChange(ChangeRecord{
		Timestamp: t1,
		Resource:  "deployment/default/payment-api",
		Namespace: "default",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})

	engine.AddChange(ChangeRecord{
		Timestamp: t2,
		Resource:  "configmap/default/payment-config",
		Namespace: "default",
		Category:  ChangeCategoryConfig,
		Summary:   "environment variable changed",
	})

	engine.AddChange(ChangeRecord{
		Timestamp: t2,
		Resource:  "service/redis/redis-svc",
		Namespace: "redis",
		Category:  ChangeCategoryKubernetes,
		Summary:   "Redis Service modified",
	})

	// Affected resource in incident is payment-api in default namespace
	affectedResources := []string{"pod/default/payment-api-7b89-xyz"}
	affectedNamespaces := []string{"default"}

	sorted := engine.Correlate(affectedResources, affectedNamespaces)
	if len(sorted) != 3 {
		t.Fatalf("Expected 3 changes, got %d", len(sorted))
	}

	top := engine.MostRelevant()
	if top == nil {
		t.Fatalf("Expected most relevant change, got nil")
	}

	if !strings.Contains(top.Summary, "payment-api v1.43") {
		t.Errorf("Expected most relevant to be payment-api v1.43, got: %s", top.Summary)
	}

	if !top.Correlated {
		t.Errorf("Expected payment-api to be correlated with affected resource")
	}

	// Verify Redis service in different namespace has lower relevance than payment-api
	var redisChange ChangeRecord
	for _, c := range sorted {
		if strings.Contains(c.Resource, "redis") {
			redisChange = c
		}
	}
	if redisChange.Relevance >= top.Relevance {
		t.Errorf("Expected Redis relevance (%.2f) to be lower than payment-api (%.2f)",
			redisChange.Relevance, top.Relevance)
	}
}

func TestChangeEngine_IngestEvidenceAndTimeline(t *testing.T) {
	refTime, _ := time.Parse("15:04", "14:05")
	engine := NewChangeEngine(60*time.Minute, refTime)

	t1, _ := time.Parse("15:04", "14:01")
	engine.AddChange(ChangeRecord{
		Timestamp: t1,
		Resource:  "deployment/default/payment-api",
		Namespace: "default",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})

	engine.Correlate([]string{"deployment/default/payment-api"}, []string{"default"})

	ec := NewEvidenceCollector()
	evList := engine.IngestEvidence(ec, 0.50)

	if len(evList) != 1 {
		t.Fatalf("Expected 1 evidence item, got %d", len(evList))
	}
	if evList[0].Source != "what-changed" {
		t.Errorf("Expected evidence source 'what-changed', got '%s'", evList[0].Source)
	}
	if !strings.Contains(evList[0].Content, "payment-api v1.43 deployed") {
		t.Errorf("Evidence content missing summary: %s", evList[0].Content)
	}

	timeline := NewTimelineEngine()
	engine.IngestTimeline(timeline, 0.50)

	events := timeline.Events()
	if len(events) != 1 {
		t.Fatalf("Expected 1 timeline event, got %d", len(events))
	}
	if events[0].Type != "change_event" {
		t.Errorf("Expected timeline event type 'change_event', got '%s'", events[0].Type)
	}
}

func TestChangeEngine_FormatForAI(t *testing.T) {
	refTime, _ := time.Parse("15:04", "14:05")
	engine := NewChangeEngine(60*time.Minute, refTime)

	t1, _ := time.Parse("15:04", "14:01")
	engine.AddChange(ChangeRecord{
		Timestamp: t1,
		Resource:  "deployment/default/payment-api",
		Namespace: "default",
		Category:  ChangeCategoryDeployment,
		Summary:   "payment-api v1.43 deployed",
	})

	engine.Correlate([]string{"payment-api"}, []string{"default"})

	prompt := engine.FormatForAI(5)
	if !strings.Contains(prompt, "payment-api v1.43 deployed") {
		t.Errorf("Prompt missing change summary: %s", prompt)
	}
	if !strings.Contains(prompt, "Most relevant change: payment-api v1.43 deployed") {
		t.Errorf("Prompt missing most relevant change callout: %s", prompt)
	}
	if !strings.Contains(prompt, "Correlation does NOT automatically imply causation") {
		t.Errorf("Prompt missing critical causality directive: %s", prompt)
	}
}

func TestChangeRecord_JSONSerialization(t *testing.T) {
	rec := ChangeRecord{
		ID:         "CHG-1",
		Timestamp:  time.Now().UTC().Truncate(time.Second),
		Resource:   "deployment/default/payment-api",
		Namespace:  "default",
		Category:   ChangeCategoryDeployment,
		Summary:    "payment-api v1.43 deployed",
		Relevance:  0.95,
		Correlated: true,
	}

	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("Failed to marshal ChangeRecord: %v", err)
	}

	var decoded ChangeRecord
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ChangeRecord: %v", err)
	}

	if decoded.ID != rec.ID || decoded.Summary != rec.Summary || !decoded.Correlated {
		t.Errorf("Decoded mismatch: %+v", decoded)
	}
}
