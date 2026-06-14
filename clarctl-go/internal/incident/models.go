// Package incident defines all structured data models for the AI-SRE engine.
package incident

import (
	"fmt"
	"math/rand"
	"time"
)

// Severity levels aligned to Google/PagerDuty SRE conventions.
type Severity string

const (
	SEV1 Severity = "SEV1" // Critical — service down, data loss risk
	SEV2 Severity = "SEV2" // High — major degradation
	SEV3 Severity = "SEV3" // Medium — partial degradation
	SEV4 Severity = "SEV4" // Low — no user impact
)

// Status lifecycle of an incident.
type Status string

const (
	StatusOpen          Status = "OPEN"
	StatusInvestigating Status = "INVESTIGATING"
	StatusMitigated     Status = "MITIGATED"
	StatusResolved      Status = "RESOLVED"
	StatusIgnored       Status = "IGNORED"
)

// RemediationStep is a single actionable step in a remediation plan.
type RemediationStep struct {
	StepNumber  int        `json:"step_number"`
	Description string     `json:"description"`
	Command     string     `json:"command,omitempty"`
	Destructive bool       `json:"is_destructive"`
	Automated   bool       `json:"is_automated"`
	Status      string     `json:"status"` // PENDING / APPLIED / SKIPPED / FAILED
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
	Result      string     `json:"result,omitempty"`
}

// ServiceImpact captures the blast radius on a specific workload.
type ServiceImpact struct {
	ServiceName  string   `json:"service_name"`
	Namespace    string   `json:"namespace"`
	ImpactLevel  string   `json:"impact_level"` // "down" / "degraded" / "at_risk"
	AffectedPods []string `json:"affected_pods,omitempty"`
}

// Report is the full structured output of one AI-SRE scan cycle.
type Report struct {
	// Identity
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Classification
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Category string   `json:"category"` // crashloop, oom, high_cpu, etc.
	Status   Status   `json:"status"`
	HasIssue bool     `json:"has_issue"`

	// Scope
	AffectedNamespaces []string        `json:"affected_namespaces"`
	AffectedServices   []ServiceImpact `json:"affected_services"`

	// Analysis
	RootCause           string   `json:"root_cause"`
	ContributingFactors []string `json:"contributing_factors"`
	ConfidenceScore     int      `json:"confidence_score"` // 0–100

	// Remediation
	RemediationPlan []RemediationStep `json:"remediation_plan"`
	RunbookUsed     string            `json:"runbook_used,omitempty"`
	AutoRemediated  bool              `json:"auto_remediated"`

	// Timing
	DetectedAt  time.Time  `json:"detected_at"`
	MitigatedAt *time.Time `json:"mitigated_at,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	MTTRSeconds *int       `json:"mttr_seconds,omitempty"`

	// Metadata
	LLMModel         string  `json:"llm_model"`
	ScanDurationSecs float64 `json:"scan_duration_seconds"`
	RawAgentOutput   string  `json:"raw_agent_output,omitempty"`
}

// NewReport creates a new Report with a unique INC- ID.
func NewReport() *Report {
	now := time.Now().UTC()
	return &Report{
		ID:         newIncidentID(),
		CreatedAt:  now,
		UpdatedAt:  now,
		DetectedAt: now,
		Status:     StatusOpen,
	}
}

// ComputeMTTR sets MTTRSeconds if the incident has been resolved or mitigated.
func (r *Report) ComputeMTTR() {
	var end *time.Time
	if r.ResolvedAt != nil {
		end = r.ResolvedAt
	} else if r.MitigatedAt != nil {
		end = r.MitigatedAt
	}
	if end != nil {
		secs := int(end.Sub(r.DetectedAt).Seconds())
		r.MTTRSeconds = &secs
	}
}

// SeverityColor returns an ANSI color code for terminal rendering.
func (r *Report) SeverityColor() string {
	switch r.Severity {
	case SEV1:
		return "\033[1;31m" // bold red
	case SEV2:
		return "\033[1;33m" // bold yellow/orange
	case SEV3:
		return "\033[1;34m" // bold blue
	default:
		return "\033[1;32m" // bold green
	}
}

// ClusterSnapshot is a point-in-time health summary persisted for trend tracking.
type ClusterSnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	TotalNodes    int       `json:"total_nodes"`
	ReadyNodes    int       `json:"ready_nodes"`
	TotalPods     int       `json:"total_pods"`
	RunningPods   int       `json:"running_pods"`
	PendingPods   int       `json:"pending_pods"`
	FailedPods    int       `json:"failed_pods"`
	CrashloopPods int       `json:"crashloop_pods"`
	CPUUsagePct   float64   `json:"cpu_usage_pct"`
	MemUsagePct   float64   `json:"mem_usage_pct"`
	OpenIncidents int       `json:"open_incidents"`
	HealthScore   float64   `json:"health_score"`
}

// ComputeHealthScore calculates a 0–100 score heuristically.
func (s *ClusterSnapshot) ComputeHealthScore() {
	score := 100.0
	if s.TotalNodes > 0 {
		nodeHealth := float64(s.ReadyNodes) / float64(s.TotalNodes) * 100
		score -= (100 - nodeHealth) * 0.4
	}
	if s.TotalPods > 0 {
		podHealth := float64(s.RunningPods) / float64(s.TotalPods) * 100
		score -= (100 - podHealth) * 0.3
	}
	score -= min64(s.CPUUsagePct*0.1, 15)
	score -= min64(s.MemUsagePct*0.1, 15)
	score -= float64(s.OpenIncidents) * 5
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	s.HealthScore = score
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func newIncidentID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("INC-%s", string(b))
}
