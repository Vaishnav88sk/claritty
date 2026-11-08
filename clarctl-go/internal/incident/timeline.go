// Package incident defines all structured data models for the AI-SRE engine.
package incident

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TimelineEvent represents a single chronological event in an incident timeline.
// It complies with the Incident Timeline Engine specification (Issue #50).
type TimelineEvent struct {
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`                // k8s-events, logs, metrics, deployments, git, argocd, ai-sre
	Type       string                 `json:"type"`                  // event, metric, log, deployment, alert, state_change
	Resource   string                 `json:"resource"`              // e.g. "deployment/redis", "pod/payment-api"
	Message    string                 `json:"message"`               // Descriptive message, e.g. "Redis endpoints → 0"
	Severity   string                 `json:"severity"`              // info, warning, critical
	EvidenceID string                 `json:"evidence_id,omitempty"` // Link to normalized Evidence ID (e.g. "E1")
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// String returns a human-readable representation of a timeline event.
func (te TimelineEvent) String() string {
	timeStr := "??:??"
	if !te.Timestamp.IsZero() {
		timeStr = te.Timestamp.Format("15:04:05")
	}

	sourceStr := ""
	if te.Source != "" {
		sourceStr = fmt.Sprintf("[%s] ", te.Source)
	}

	resStr := ""
	if te.Resource != "" {
		resStr = fmt.Sprintf("%s: ", te.Resource)
	}

	evRef := ""
	if te.EvidenceID != "" {
		evRef = fmt.Sprintf(" (%s)", te.EvidenceID)
	}

	return fmt.Sprintf("%s  %s%s%s%s", timeStr, sourceStr, resStr, te.Message, evRef)
}

// TimelineFilter specifies criteria for querying timeline events.
type TimelineFilter struct {
	StartTime   time.Time
	EndTime     time.Time
	Sources     []string
	Types       []string
	Resource    string
	MinSeverity string
}

// TimelineEngine collects, correlates, orders, and queries timeline events from multiple sources.
type TimelineEngine struct {
	events []TimelineEvent
}

// NewTimelineEngine creates a new empty timeline engine.
func NewTimelineEngine() *TimelineEngine {
	return &TimelineEngine{
		events: make([]TimelineEvent, 0),
	}
}

// Add appends a timeline event.
func (te *TimelineEngine) Add(e TimelineEvent) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	te.events = append(te.events, e)
}

// AddMany appends multiple timeline events.
func (te *TimelineEngine) AddMany(events ...TimelineEvent) {
	for _, e := range events {
		te.Add(e)
	}
}

// IngestEvidence converts a normalized Evidence item into a TimelineEvent and adds it.
func (te *TimelineEngine) IngestEvidence(ev Evidence) TimelineEvent {
	evtType := string(ev.Type)
	if evtType == "" {
		evtType = "telemetry"
	}

	evt := TimelineEvent{
		Timestamp:  ev.Timestamp,
		Source:     ev.Source,
		Type:       evtType,
		Resource:   ev.Resource,
		Message:    ev.Content,
		Severity:   string(ev.Severity),
		EvidenceID: ev.ID,
		Metadata:   ev.Metadata,
	}
	te.Add(evt)
	return evt
}

// Events returns all timeline events sorted chronologically.
// Events with valid timestamps are ordered earliest to latest.
// Events with missing/zero timestamps are placed at the end in deterministic order.
func (te *TimelineEngine) Events() []TimelineEvent {
	out := make([]TimelineEvent, len(te.events))
	copy(out, te.events)

	sort.SliceStable(out, func(i, j int) bool {
		ti, tj := out[i].Timestamp, out[j].Timestamp
		zeroI, zeroJ := ti.IsZero(), tj.IsZero()

		if zeroI && zeroJ {
			// Both timestamps missing: fallback to source/resource comparison
			if out[i].Source != out[j].Source {
				return out[i].Source < out[j].Source
			}
			return out[i].Message < out[j].Message
		}
		if zeroI {
			return false // Push zero timestamp to the end
		}
		if zeroJ {
			return true // Valid timestamp comes before zero timestamp
		}
		return ti.Before(tj)
	})

	return out
}

// Query filters the timeline by specific criteria while preserving chronological order.
func (te *TimelineEngine) Query(filter TimelineFilter) []TimelineEvent {
	var result []TimelineEvent
	sourceMap := make(map[string]bool)
	for _, s := range filter.Sources {
		sourceMap[strings.ToLower(s)] = true
	}

	typeMap := make(map[string]bool)
	for _, t := range filter.Types {
		typeMap[strings.ToLower(t)] = true
	}

	for _, e := range te.Events() {
		if !filter.StartTime.IsZero() && !e.Timestamp.IsZero() && e.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && !e.Timestamp.IsZero() && e.Timestamp.After(filter.EndTime) {
			continue
		}
		if len(sourceMap) > 0 && !sourceMap[strings.ToLower(e.Source)] {
			continue
		}
		if len(typeMap) > 0 && !typeMap[strings.ToLower(e.Type)] {
			continue
		}
		if filter.Resource != "" && !strings.Contains(strings.ToLower(e.Resource), strings.ToLower(filter.Resource)) {
			continue
		}
		if filter.MinSeverity != "" && !matchesMinSeverity(e.Severity, filter.MinSeverity) {
			continue
		}
		result = append(result, e)
	}

	return result
}

// Between returns events within the specified time window.
func (te *TimelineEngine) Between(start, end time.Time) []TimelineEvent {
	return te.Query(TimelineFilter{
		StartTime: start,
		EndTime:   end,
	})
}

// BySource returns all events from a given source.
func (te *TimelineEngine) BySource(source string) []TimelineEvent {
	return te.Query(TimelineFilter{
		Sources: []string{source},
	})
}

// ByResource returns all events related to a specific resource.
func (te *TimelineEngine) ByResource(resource string) []TimelineEvent {
	return te.Query(TimelineFilter{
		Resource: resource,
	})
}

// FormatForAI renders a concise, chronological text view for LLM prompt context.
func (te *TimelineEngine) FormatForAI(maxEvents int) string {
	events := te.Events()
	if len(events) == 0 {
		return "(No timeline events recorded)"
	}

	if maxEvents > 0 && len(events) > maxEvents {
		events = events[:maxEvents]
	}

	var sb strings.Builder
	for _, e := range events {
		timeStr := "unknown-time"
		if !e.Timestamp.IsZero() {
			timeStr = e.Timestamp.Format("15:04:05")
		}

		evRef := ""
		if e.EvidenceID != "" {
			evRef = fmt.Sprintf(" [%s]", e.EvidenceID)
		}

		resStr := ""
		if e.Resource != "" {
			resStr = fmt.Sprintf(" %s:", e.Resource)
		}

		sb.WriteString(fmt.Sprintf("%s [%s]%s %s%s\n", timeStr, e.Source, resStr, e.Message, evRef))
	}

	return strings.TrimSpace(sb.String())
}

// ToJSON serializes the timeline to a JSON string.
func (te *TimelineEngine) ToJSON() (string, error) {
	b, err := json.Marshal(te.Events())
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func matchesMinSeverity(actual, min string) bool {
	rank := func(s string) int {
		switch strings.ToLower(s) {
		case "critical":
			return 3
		case "warning":
			return 2
		case "info":
			return 1
		default:
			return 0
		}
	}
	return rank(actual) >= rank(min)
}
