// Package incident defines all structured data models for the AI-SRE engine.
package incident

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EvidenceType represents the category of telemetry or system state collected.
type EvidenceType string

const (
	EvidenceTypeMetric   EvidenceType = "metric"
	EvidenceTypeLog      EvidenceType = "log"
	EvidenceTypeEvent    EvidenceType = "event"
	EvidenceTypeResource EvidenceType = "resource"
	EvidenceTypeConfig   EvidenceType = "config"
)

// EvidenceSeverity indicates the severity level of an observed piece of evidence.
type EvidenceSeverity string

const (
	EvidenceSeverityInfo     EvidenceSeverity = "info"
	EvidenceSeverityWarning  EvidenceSeverity = "warning"
	EvidenceSeverityCritical EvidenceSeverity = "critical"
)

// Evidence represents a normalized unit of factual telemetry or cluster observation.
// It complies with the Unified Evidence Model (Issue #48).
type Evidence struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Type      EvidenceType           `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Resource  string                 `json:"resource"`
	Severity  EvidenceSeverity       `json:"severity"`
	Content   string                 `json:"content"`
	Metric    string                 `json:"metric,omitempty"`
	Value     *float64               `json:"value,omitempty"`
	Relevance float64                `json:"relevance,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// String returns a clean, human-readable summary of the evidence.
func (e Evidence) String() string {
	var parts []string
	if e.ID != "" {
		parts = append(parts, e.ID)
	}
	if e.Resource != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Resource))
	}
	parts = append(parts, e.Content)
	if e.Metric != "" && e.Value != nil {
		parts = append(parts, fmt.Sprintf("(metric: %s=%.2f)", e.Metric, *e.Value))
	}
	return strings.Join(parts, " - ")
}

// Conclusion represents a root cause or diagnostic finding backed by explicit evidence IDs.
type Conclusion struct {
	Statement   string   `json:"statement"`
	EvidenceIDs []string `json:"evidence_ids"`
	Reasoning   string   `json:"reasoning"`
}

// Hypothesis represents an unverified potential cause that lacks direct evidence.
type Hypothesis struct {
	Statement       string   `json:"statement"`
	Reason          string   `json:"reason"`
	ValidationSteps []string `json:"validation_steps,omitempty"`
}

// EvidenceCollector manages evidence collection, indexing, and prompt formatting.
type EvidenceCollector struct {
	evidence []Evidence
	byID     map[string]Evidence
	counter  int
}

// NewEvidenceCollector creates an empty collector.
func NewEvidenceCollector() *EvidenceCollector {
	return &EvidenceCollector{
		evidence: make([]Evidence, 0),
		byID:     make(map[string]Evidence),
	}
}

// NextID returns the next sequential evidence identifier (e.g., E1, E2, ...).
func (ec *EvidenceCollector) NextID() string {
	ec.counter++
	return fmt.Sprintf("E%d", ec.counter)
}

// Add appends a piece of evidence, auto-assigning an ID if missing.
func (ec *EvidenceCollector) Add(ev Evidence) Evidence {
	if ev.ID == "" {
		ev.ID = ec.NextID()
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Metadata == nil {
		ev.Metadata = make(map[string]interface{})
	}
	ec.evidence = append(ec.evidence, ev)
	ec.byID[ev.ID] = ev
	return ev
}

// All returns all collected evidence items.
func (ec *EvidenceCollector) All() []Evidence {
	out := make([]Evidence, len(ec.evidence))
	copy(out, ec.evidence)
	return out
}

// GetByID finds an evidence item by its ID.
func (ec *EvidenceCollector) GetByID(id string) (Evidence, bool) {
	ev, ok := ec.byID[id]
	return ev, ok
}

// FormatForPrompt renders a structured, numbered list of evidence for LLM prompt ingestion.
func (ec *EvidenceCollector) FormatForPrompt() string {
	if len(ec.evidence) == 0 {
		return "(No explicit telemetry evidence recorded)"
	}
	var sb strings.Builder
	for _, ev := range ec.evidence {
		sb.WriteString(fmt.Sprintf("%s [%s/%s] (%s) %s: %s\n",
			ev.ID, ev.Type, ev.Source, ev.Severity, ev.Resource, ev.Content))
	}
	return strings.TrimSpace(sb.String())
}

// ToJSON serializes the evidence list to a JSON string.
func (ec *EvidenceCollector) ToJSON() (string, error) {
	b, err := json.Marshal(ec.evidence)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

