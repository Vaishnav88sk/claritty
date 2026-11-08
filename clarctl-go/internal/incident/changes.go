// Package incident defines all structured data models for the AI-SRE engine.
package incident

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ChangeCategory identifies the domain of a detected change.
type ChangeCategory string

const (
	ChangeCategoryDeployment  ChangeCategory = "deployment"
	ChangeCategoryConfig      ChangeCategory = "config"
	ChangeCategoryGitCommit   ChangeCategory = "git_commit"
	ChangeCategoryImage       ChangeCategory = "image"
	ChangeCategoryEnvironment ChangeCategory = "environment"
	ChangeCategoryKubernetes  ChangeCategory = "kubernetes"
	ChangeCategoryArgoCD      ChangeCategory = "argocd"
)

// ChangeRecord represents a discrete configuration, deployment, or environment change.
// Complies with Issue #51 ("What Changed?" Investigation).
type ChangeRecord struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Resource   string                 `json:"resource"` // e.g. "deployment/default/payment-api"
	Namespace  string                 `json:"namespace,omitempty"`
	Category   ChangeCategory         `json:"category"`
	Summary    string                 `json:"summary"` // e.g. "payment-api v1.43 deployed"
	Details    map[string]interface{} `json:"details,omitempty"`
	Relevance  float64                `json:"relevance"`  // 0.0 - 1.0 correlation score
	Correlated bool                   `json:"correlated"` // true if correlated with affected resources
	EvidenceID string                 `json:"evidence_id,omitempty"`
}

// String returns a clean, human-readable summary of the change.
func (c ChangeRecord) String() string {
	timeStr := "Unknown time"
	if !c.Timestamp.IsZero() {
		timeStr = c.Timestamp.UTC().Format("15:04")
	}
	res := c.Resource
	if res == "" && c.Namespace != "" {
		res = c.Namespace
	}
	rel := ""
	if c.Relevance > 0 {
		rel = fmt.Sprintf(" (relevance: %.2f)", c.Relevance)
	}
	return fmt.Sprintf("[%s] %s: %s%s", timeStr, res, c.Summary, rel)
}

// ChangeEngine manages collection, correlation, and evidence generation for changes.
type ChangeEngine struct {
	windowDuration time.Duration
	referenceTime  time.Time
	changes        []ChangeRecord
	counter        int
}

// NewChangeEngine creates a ChangeEngine with a lookback window and reference time.
func NewChangeEngine(window time.Duration, refTime time.Time) *ChangeEngine {
	if window <= 0 {
		window = 60 * time.Minute
	}
	if refTime.IsZero() {
		refTime = time.Now().UTC()
	}
	return &ChangeEngine{
		windowDuration: window,
		referenceTime:  refTime,
		changes:        make([]ChangeRecord, 0),
	}
}

// NextID generates a unique identifier for a change record (e.g., CHG-1, CHG-2).
func (ce *ChangeEngine) NextID() string {
	ce.counter++
	return fmt.Sprintf("CHG-%d", ce.counter)
}

// AddChange adds a change record if it falls within the configured lookback window.
// Returns true if the change was accepted within the window.
func (ce *ChangeEngine) AddChange(rec ChangeRecord) bool {
	windowStart := ce.referenceTime.Add(-ce.windowDuration)
	// Accept if timestamp is zero or within [windowStart, referenceTime + 5m]
	if !rec.Timestamp.IsZero() {
		if rec.Timestamp.Before(windowStart) || rec.Timestamp.After(ce.referenceTime.Add(5*time.Minute)) {
			return false
		}
	} else {
		rec.Timestamp = ce.referenceTime
	}

	if rec.ID == "" {
		rec.ID = ce.NextID()
	}
	if rec.Details == nil {
		rec.Details = make(map[string]interface{})
	}

	ce.changes = append(ce.changes, rec)
	return true
}

// All returns all recorded changes.
func (ce *ChangeEngine) All() []ChangeRecord {
	out := make([]ChangeRecord, len(ce.changes))
	copy(out, ce.changes)
	return out
}

// Correlate evaluates each change against affected resources and namespaces,
// computing a correlation relevance score (0.0 to 1.0).
func (ce *ChangeEngine) Correlate(affectedResources []string, affectedNamespaces []string) []ChangeRecord {
	affResNorm := make([]string, 0, len(affectedResources))
	for _, r := range affectedResources {
		affResNorm = append(affResNorm, strings.ToLower(r))
	}

	affNsMap := make(map[string]bool)
	for _, ns := range affectedNamespaces {
		affNsMap[strings.ToLower(ns)] = true
	}

	for i := range ce.changes {
		c := &ce.changes[i]

		// 1. Proximity score (0.4 to 1.0)
		timeWeight := 0.70
		if !c.Timestamp.IsZero() {
			delta := ce.referenceTime.Sub(c.Timestamp)
			if delta < 0 {
				delta = -delta
			}
			switch {
			case delta <= 5*time.Minute:
				timeWeight = 1.0
			case delta <= 15*time.Minute:
				timeWeight = 0.90
			case delta <= 30*time.Minute:
				timeWeight = 0.75
			case delta <= 60*time.Minute:
				timeWeight = 0.60
			default:
				timeWeight = 0.40
			}
		}

		// 2. Resource & Namespace match score (0.2 to 1.0)
		resourceWeight := 0.30
		resLower := strings.ToLower(c.Resource)
		nsLower := strings.ToLower(c.Namespace)

		matchedWorkload := false
		for _, aff := range affResNorm {
			if aff == "" {
				continue
			}
			// Check if resource name or parts match (e.g. "payment-api" matches "pod/default/payment-api-abc")
			baseAff := aff
			if idx := strings.LastIndex(baseAff, "/"); idx != -1 {
				baseAff = baseAff[idx+1:]
			}
			baseRes := resLower
			if idx := strings.LastIndex(baseRes, "/"); idx != -1 {
				baseRes = baseRes[idx+1:]
			}

			if strings.Contains(resLower, baseAff) || strings.Contains(aff, baseRes) {
				matchedWorkload = true
				break
			}
		}

		if matchedWorkload {
			resourceWeight = 1.0
		} else if affNsMap[nsLower] || nsLower == "" {
			resourceWeight = 0.60
		}

		// 3. Category weight (0.8 to 1.0)
		categoryWeight := 0.85
		switch c.Category {
		case ChangeCategoryDeployment, ChangeCategoryImage, ChangeCategoryConfig, ChangeCategoryArgoCD:
			categoryWeight = 1.0
		case ChangeCategoryGitCommit:
			categoryWeight = 0.95
		case ChangeCategoryKubernetes:
			categoryWeight = 0.85
		case ChangeCategoryEnvironment:
			categoryWeight = 0.80
		}

		score := timeWeight * resourceWeight * categoryWeight
		if score > 1.0 {
			score = 1.0
		}
		c.Relevance = math.Round(score*100) / 100
		c.Correlated = matchedWorkload || (c.Relevance >= 0.50)
	}

	// Sort changes: correlated first, then by relevance descending, then by timestamp descending
	sort.SliceStable(ce.changes, func(i, j int) bool {
		if ce.changes[i].Correlated != ce.changes[j].Correlated {
			return ce.changes[i].Correlated
		}
		if ce.changes[i].Relevance != ce.changes[j].Relevance {
			return ce.changes[i].Relevance > ce.changes[j].Relevance
		}
		return ce.changes[i].Timestamp.After(ce.changes[j].Timestamp)
	})

	return ce.All()
}

// IngestEvidence converts correlated changes into Evidence items and registers them in EvidenceCollector.
func (ce *ChangeEngine) IngestEvidence(ec *EvidenceCollector, minRelevance float64) []Evidence {
	if ec == nil {
		return nil
	}
	var out []Evidence
	for i := range ce.changes {
		c := &ce.changes[i]
		if c.Relevance < minRelevance && !c.Correlated {
			continue
		}

		evType := EvidenceTypeConfig
		if c.Category == ChangeCategoryDeployment || c.Category == ChangeCategoryImage {
			evType = EvidenceTypeResource
		} else if c.Category == ChangeCategoryEnvironment || c.Category == ChangeCategoryKubernetes {
			evType = EvidenceTypeEvent
		}

		sev := EvidenceSeverityInfo
		if c.Correlated && c.Relevance >= 0.80 {
			sev = EvidenceSeverityWarning
		}

		content := fmt.Sprintf("[What Changed?] %s (category: %s)", c.Summary, c.Category)
		meta := make(map[string]interface{})
		for k, v := range c.Details {
			meta[k] = v
		}
		meta["change_id"] = c.ID
		meta["category"] = string(c.Category)

		ev := ec.Add(Evidence{
			Source:    "what-changed",
			Type:      evType,
			Timestamp: c.Timestamp,
			Resource:  c.Resource,
			Severity:  sev,
			Content:   content,
			Relevance: c.Relevance,
			Metadata:  meta,
		})

		c.EvidenceID = ev.ID
		out = append(out, ev)
	}
	return out
}

// IngestTimeline adds correlated changes into the timeline engine chronologically.
func (ce *ChangeEngine) IngestTimeline(timeline *TimelineEngine, minRelevance float64) {
	if timeline == nil {
		return
	}
	for _, c := range ce.changes {
		if c.Relevance < minRelevance && !c.Correlated {
			continue
		}
		sev := "info"
		if c.Correlated && c.Relevance >= 0.80 {
			sev = "warning"
		}
		timeline.Add(TimelineEvent{
			Timestamp:  c.Timestamp,
			Source:     fmt.Sprintf("change:%s", c.Category),
			Type:       "change_event",
			Resource:   c.Resource,
			Message:    c.Summary,
			Severity:   sev,
			EvidenceID: c.EvidenceID,
		})
	}
}

// MostRelevant returns the change record with the highest correlation score, if any.
func (ce *ChangeEngine) MostRelevant() *ChangeRecord {
	if len(ce.changes) == 0 {
		return nil
	}
	// Changes are already sorted by Correlated & Relevance descending
	top := ce.changes[0]
	if top.Relevance > 0 {
		return &top
	}
	return nil
}

// FormatForAI formats the detected changes for LLM prompts with clear causality instructions.
func (ce *ChangeEngine) FormatForAI(maxItems int) string {
	if len(ce.changes) == 0 {
		return "(No significant changes detected in the lookback window)"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("POTENTIAL CHANGES DETECTED (Window: %s):\n", ce.windowDuration.String()))

	count := 0
	for _, c := range ce.changes {
		if count >= maxItems {
			break
		}
		timeStr := "Unknown time"
		if !c.Timestamp.IsZero() {
			timeStr = c.Timestamp.UTC().Format("15:04")
		}
		corrStr := "uncorrelated"
		if c.Correlated {
			corrStr = "CORRELATED with affected resources"
		}
		fmt.Fprintf(&sb, "• [%s] %s: %s (Category: %s, Relevance: %.2f, %s)\n",
			timeStr, c.Resource, c.Summary, c.Category, c.Relevance, corrStr)
		count++
	}

	if top := ce.MostRelevant(); top != nil && top.Correlated {
		fmt.Fprintf(&sb, "\nMost relevant change: %s (%s)\n", top.Summary, top.Resource)
	}

	sb.WriteString("\nCRITICAL CAUSALITY DIRECTIVE:\n")
	sb.WriteString("Correlation does NOT automatically imply causation. A recent change may be coincidental.\n")
	sb.WriteString("Only conclude that a change caused the incident if logs, error stacktraces, or metrics directly connect the failure to the change. Otherwise, list it as a hypothesis or contributing factor.\n")

	return sb.String()
}
