package incident

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAgentTimelineEngine_OrderingAndMissingTimestamps(t *testing.T) {
	engine := NewTimelineEngine()
	baseTime := time.Date(2026, 11, 8, 14, 0, 0, 0, time.UTC)

	// Ingest events out of order, including one missing timestamp
	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(5 * time.Minute),
		Source:    "metrics",
		Message:   "Error rate spike",
	})

	engine.Add(TimelineEvent{
		Timestamp: baseTime.Add(2 * time.Minute),
		Source:    "deployments",
		Message:   "Deploy v1.2",
	})

	engine.Add(TimelineEvent{
		Timestamp: time.Time{},
		Source:    "git",
		Message:   "Commit without timestamp",
	})

	events := engine.Events()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	if events[0].Message != "Deploy v1.2" {
		t.Errorf("Expected earliest event first, got: %s", events[0].Message)
	}
	if events[1].Message != "Error rate spike" {
		t.Errorf("Expected second event, got: %s", events[1].Message)
	}
	if !events[2].Timestamp.IsZero() || events[2].Message != "Commit without timestamp" {
		t.Errorf("Expected missing timestamp event last, got: %+v", events[2])
	}
}

func TestAgentTimelineEngine_QueryAndSerialization(t *testing.T) {
	engine := NewTimelineEngine()
	now := time.Now().UTC()

	engine.Add(TimelineEvent{
		Timestamp: now,
		Source:    "k8s-events",
		Resource:  "pod/redis",
		Message:   "BackOff",
		Severity:  "warning",
	})

	res := engine.BySource("k8s-events")
	if len(res) != 1 {
		t.Fatalf("Expected 1 event from k8s-events, got %d", len(res))
	}

	jsonStr, err := engine.ToJSON()
	if err != nil || !strings.Contains(jsonStr, "BackOff") {
		t.Fatalf("JSON serialization failed: %v, %s", err, jsonStr)
	}

	var decoded []TimelineEvent
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}
	if len(decoded) != 1 {
		t.Errorf("Expected 1 decoded event, got %d", len(decoded))
	}
}
