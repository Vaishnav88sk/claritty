// Package incident defines the shared incident report data types.
package incident

import (
	"fmt"
	"time"
)

type Severity string
type Status string

const (
	SEV1 Severity = "SEV1"
	SEV2 Severity = "SEV2"
	SEV3 Severity = "SEV3"
	SEV4 Severity = "SEV4"
)

const (
	StatusInvestigating Status = "INVESTIGATING"
	StatusMitigated     Status = "MITIGATED"
	StatusResolved      Status = "RESOLVED"
)

type AffectedService struct {
	ServiceName string `json:"service_name"`
	Namespace   string `json:"namespace"`
	ImpactLevel string `json:"impact_level"` // down | degraded | at_risk
}

type RemediationStep struct {
	StepNumber  int    `json:"step_number"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Destructive bool   `json:"is_destructive"`
	Automated   bool   `json:"is_automated"`
}

type ClusterSnapshot struct {
	ClusterName   string    `json:"cluster_name"`
	HealthScore   float64   `json:"health_score"`
	TotalNodes    int       `json:"total_nodes"`
	ReadyNodes    int       `json:"ready_nodes"`
	RunningPods   int       `json:"running_pods"`
	PendingPods   int       `json:"pending_pods"`
	FailedPods    int       `json:"failed_pods"`
	CrashloopPods int       `json:"crashloop_pods"`
	Namespaces    []string  `json:"namespaces"`
	CapturedAt    time.Time `json:"captured_at"`
}

type Report struct {
	ID                  string            `json:"id"`
	ClusterName         string            `json:"cluster_name"`
	HasIssue            bool              `json:"has_issue"`
	Severity            Severity          `json:"severity"`
	Status              Status            `json:"status"`
	Title               string            `json:"title"`
	Category            string            `json:"category"`
	AffectedNamespaces  []string          `json:"affected_namespaces"`
	AffectedServices    []AffectedService `json:"affected_services"`
	RootCause           string            `json:"root_cause"`
	ContributingFactors []string          `json:"contributing_factors"`
	ConfidenceScore     int               `json:"confidence_score"`
	RemediationPlan     []RemediationStep `json:"remediation_plan"`
	RunbookUsed         string            `json:"runbook_used"`
	LLMModel            string            `json:"llm_model"`
	DetectedAt          time.Time         `json:"detected_at"`
	ResolvedAt          *time.Time        `json:"resolved_at,omitempty"`
	ScanDurationSecs    float64           `json:"scan_duration_secs"`
	ClusterSnapshot     *ClusterSnapshot  `json:"cluster_snapshot,omitempty"`
}

func NewReport(clusterName string) *Report {
	return &Report{
		ID:          generateID(),
		ClusterName: clusterName,
		Status:      StatusInvestigating,
		Severity:    SEV4,
		DetectedAt:  time.Now().UTC(),
	}
}

func generateID() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	t := time.Now().UnixNano()
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[t%int64(len(letters))]
		t /= int64(len(letters))
		if t == 0 {
			t = time.Now().UnixNano() + int64(i*997)
		}
	}
	return fmt.Sprintf("INC-%s", string(b))
}
