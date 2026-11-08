// Package ai implements the LLM-powered 6-stage SRE analysis pipeline.
// It is adapted from clarctl-go/internal/ai/pipeline.go but outputs
// structured JSON to be sent to the hub server rather than printing to a terminal.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/config"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/incident"
	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/k8s"
)

const maxRetries = 3

var retryBackoffs = []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second}
var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

// Pipeline orchestrates the 6-stage SRE scan.
type Pipeline struct {
	cfg    *config.Config
	k8sCli *k8s.Client
	llm    llms.Model
}

// New creates a pipeline with an LLM and Kubernetes client.
func New(cfg *config.Config, k8sCli *k8s.Client) (*Pipeline, error) {
	llm, err := buildLLM(cfg)
	if err != nil {
		return nil, fmt.Errorf("init LLM: %w", err)
	}
	return &Pipeline{cfg: cfg, k8sCli: k8sCli, llm: llm}, nil
}

func buildLLM(cfg *config.Config) (llms.Model, error) {
	model := cfg.LLMModel
	switch cfg.LLMProvider {
	case "groq":
		model = strings.TrimPrefix(model, "groq/")
		model = strings.TrimPrefix(model, "groq/")
		return openai.New(
			openai.WithBaseURL("https://api.groq.com/openai/v1"),
			openai.WithToken(cfg.GroqAPIKey),
			openai.WithModel(model),
		)
	case "openai":
		return openai.New(
			openai.WithToken(cfg.OpenAIAPIKey),
			openai.WithModel(model),
		)
	case "mistral":
		model = strings.TrimPrefix(model, "mistral/")
		return openai.New(
			openai.WithBaseURL("https://api.mistral.ai/v1"),
			openai.WithToken(cfg.MistralAPIKey),
			openai.WithModel(model),
		)
	case "anthropic":
		model = strings.TrimPrefix(model, "anthropic/")
		return anthropic.New(
			anthropic.WithToken(cfg.AnthropicAPIKey),
			anthropic.WithModel(model),
		)
	case "ollama":
		model = strings.TrimPrefix(model, "ollama/")
		return ollama.New(
			ollama.WithServerURL(cfg.OllamaHost),
			ollama.WithModel(model),
		)
	case "mock":
		return &MockLLM{}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}
}

// RunScan executes the full pipeline and returns a structured incident report.
func (p *Pipeline) RunScan(ctx context.Context) (*incident.Report, error) {
	start := time.Now()
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		report, err := p.runPipeline(ctx)
		if err == nil {
			report.ScanDurationSecs = time.Since(start).Seconds()
			return report, nil
		}
		lastErr = err
		if isRateLimit(err) && attempt < maxRetries-1 {
			wait := retryBackoffs[attempt]
			fmt.Printf("Rate limit — retrying in %s (%d/%d)\n", wait, attempt+2, maxRetries)
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		break
	}
	return nil, lastErr
}

func (p *Pipeline) runPipeline(ctx context.Context) (*incident.Report, error) {
	ec := incident.NewEvidenceCollector()
	timeline := incident.NewTimelineEngine()

	fmt.Println("[agent] Stage 1/6: Triage — collecting cluster state...")
	triage, err := p.collectTriage(ctx, ec, timeline)
	if err != nil {
		return nil, fmt.Errorf("triage: %w", err)
	}

	fmt.Println("[agent] Stage 2/6: Metrics — analyzing resource usage...")
	metricsSummary, err := p.runAgent(ctx, buildMetricsPrompt(triage))
	if err != nil {
		return nil, fmt.Errorf("metrics: %w", err)
	}

	fmt.Println("[agent] Stage 3/6: Logs — mining error patterns...")
	logSummary, err := p.runAgent(ctx, buildLogsPrompt(triage, truncate(metricsSummary, 800)))
	if err != nil {
		return nil, fmt.Errorf("logs: %w", err)
	}

	fmt.Println("[agent] Stage 4/6: Infra — diagnosing constraints...")
	infraSummary, err := p.runAgent(ctx, buildInfraPrompt(truncate(triage, 1200), truncate(metricsSummary, 800), truncate(logSummary, 800)))
	if err != nil {
		return nil, fmt.Errorf("infra: %w", err)
	}

	fmt.Println("[agent] Stage 5/6: Runbook — selecting remediation plan...")
	runbookSummary, err := p.runAgent(ctx, buildRunbookPrompt(truncate(triage, 800), truncate(metricsSummary, 600), truncate(logSummary, 600), truncate(infraSummary, 600)))
	if err != nil {
		return nil, fmt.Errorf("runbook: %w", err)
	}

	fmt.Println("[agent] Stage 6/6: Commander — synthesizing final report...")
	rawJSON, err := p.runAgent(ctx, buildCommanderPrompt(
		truncate(triage, 1000),
		truncate(metricsSummary, 700),
		truncate(logSummary, 700),
		truncate(infraSummary, 700),
		truncate(runbookSummary, 800),
		ec,
		timeline,
	))
	if err != nil {
		return nil, fmt.Errorf("commander: %w", err)
	}

	return parseReport(rawJSON, p.cfg.ClusterName, fullModelName(p.cfg), ec, timeline)
}

// collectTriage gathers cluster data and builds the triage prompt input.
func (p *Pipeline) collectTriage(ctx context.Context, ec *incident.EvidenceCollector, timeline *incident.TimelineEngine) (string, error) {
	snap, pods, err := p.k8sCli.CollectSnapshot(ctx, p.cfg.Namespaces)
	if err != nil {
		return "", err
	}
	events := p.k8sCli.CollectEvents(ctx, "")

	var sb strings.Builder
	fmt.Fprintf(&sb, "CLUSTER: %s\nNodes: %d/%d Ready\nPods: %d Running / %d Pending / %d Failed / %d CrashLoop\n\n",
		p.cfg.ClusterName, snap.ReadyNodes, snap.TotalNodes,
		snap.RunningPods, snap.PendingPods, snap.FailedPods, snap.CrashloopPods)

	fmt.Fprintf(&sb, "PROBLEMATIC PODS:\n")
	for _, pod := range pods {
		if pod.Restarts > 2 || pod.Reason != "" || pod.Phase == "Pending" || pod.Phase == "Failed" {
			fmt.Fprintf(&sb, "  Pod=%s NS=%s Phase=%s Reason=%s Restarts=%d Node=%s\n",
				pod.Name, pod.Namespace, pod.Phase, pod.Reason, pod.Restarts, pod.Node)

			var evID string
			if ec != nil {
				sev := incident.EvidenceSeverityWarning
				if pod.Restarts > 5 || pod.Phase == "Failed" || pod.Reason == "CrashLoopBackOff" || pod.Reason == "OOMKilled" {
					sev = incident.EvidenceSeverityCritical
				}
				evItem := ec.Add(incident.Evidence{
					Source:    "kubernetes",
					Type:      incident.EvidenceTypeResource,
					Resource:  fmt.Sprintf("pod/%s/%s", pod.Namespace, pod.Name),
					Severity:  sev,
					Content:   fmt.Sprintf("Pod %s in namespace %s is in %s state (reason: %s, restarts: %d, node: %s)", pod.Name, pod.Namespace, pod.Phase, pod.Reason, pod.Restarts, pod.Node),
					Relevance: 0.90,
				})
				evID = evItem.ID
			}

			if timeline != nil {
				timeline.Add(incident.TimelineEvent{
					Timestamp:  time.Now().UTC(),
					Source:     "kubernetes",
					Type:       "pod_state",
					Resource:   fmt.Sprintf("pod/%s/%s", pod.Namespace, pod.Name),
					Message:    fmt.Sprintf("Pod %s is %s (restarts: %d)", pod.Name, pod.Phase, pod.Restarts),
					Severity:   "warning",
					EvidenceID: evID,
				})
			}

			if pod.Logs != "" {
				fmt.Fprintf(&sb, "  Logs(tail): %s\n", truncate(pod.Logs, 500))
				var logEvID string
				if ec != nil {
					evItem := ec.Add(incident.Evidence{
						Source:    "logs",
						Type:      incident.EvidenceTypeLog,
						Resource:  fmt.Sprintf("pod/%s/%s", pod.Namespace, pod.Name),
						Severity:  incident.EvidenceSeverityCritical,
						Content:   fmt.Sprintf("Pod logs: %s", truncate(pod.Logs, 150)),
						Relevance: 0.92,
					})
					logEvID = evItem.ID
				}
				if timeline != nil {
					timeline.Add(incident.TimelineEvent{
						Timestamp:  time.Now().UTC(),
						Source:     "logs",
						Type:       "log_error",
						Resource:   fmt.Sprintf("pod/%s/%s", pod.Namespace, pod.Name),
						Message:    fmt.Sprintf("Logs error: %s", truncate(pod.Logs, 120)),
						Severity:   "critical",
						EvidenceID: logEvID,
					})
				}
			}
		}
	}

	if events != "" {
		fmt.Fprintf(&sb, "\nRECENT WARNING EVENTS:\n%s", truncate(events, 1000))
		var evtID string
		if ec != nil {
			evItem := ec.Add(incident.Evidence{
				Source:    "k8s-events",
				Type:      incident.EvidenceTypeEvent,
				Resource:  p.cfg.ClusterName,
				Severity:  incident.EvidenceSeverityWarning,
				Content:   fmt.Sprintf("Warning events: %s", truncate(events, 200)),
				Relevance: 0.85,
			})
			evtID = evItem.ID
		}
		if timeline != nil {
			timeline.Add(incident.TimelineEvent{
				Timestamp:  time.Now().UTC(),
				Source:     "k8s-events",
				Type:       "warning_event",
				Resource:   p.cfg.ClusterName,
				Message:    truncate(events, 150),
				Severity:   "warning",
				EvidenceID: evtID,
			})
		}
	}

	return sb.String(), nil
}

func (p *Pipeline) runAgent(ctx context.Context, prompt string) (string, error) {
	resp, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt,
		llms.WithTemperature(0.1),
		llms.WithMaxTokens(800),
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp), nil
}

// ─── Prompts ─────────────────────────────────────────────────────────────────

func buildMetricsPrompt(triage string) string {
	return fmt.Sprintf(`You are a Kubernetes Metrics Analyst.
Analyze the cluster state below and summarize resource pressure.

CLUSTER STATE:
%s

Output a concise summary (max 200 words) covering:
1. Which pods have highest restart counts
2. Any OOMKilled or memory-pressure indicators
3. CPU pressure signs
4. Which namespaces are most affected`, truncate(triage, 1500))
}

func buildLogsPrompt(triage, metrics string) string {
	return fmt.Sprintf(`You are a Log Analysis Expert.
Based on the cluster state and metrics, identify the root error patterns.

CLUSTER STATE: %s
METRICS: %s

Output a concise summary (max 200 words) of:
1. The top error messages found in pod logs
2. Whether errors are application bugs, config issues, or infra problems
3. Which specific pods are generating errors`, triage, metrics)
}

func buildInfraPrompt(triage, metrics, logs string) string {
	return fmt.Sprintf(`You are a Kubernetes Infrastructure Diagnostician.
Diagnose if infrastructure constraints are causing the incident.

TRIAGE: %s
METRICS: %s
LOGS: %s

Analyze (max 200 words):
1. Are resource quotas being hit?
2. Are there scheduling constraints (taints, affinity)?
3. Are nodes healthy?
4. Is this infra-caused or application-caused?`, triage, metrics, logs)
}

func buildRunbookPrompt(triage, metrics, logs, infra string) string {
	return fmt.Sprintf(`You are a senior Kubernetes Runbook & Remediation Engineer.
Your job is to create an ACTIONABLE remediation plan that FIXES the problem.

TRIAGE: %s
METRICS: %s
LOGS: %s
INFRA: %s

CRITICAL RULES:
1. DO NOT suggest 'kubectl describe' or 'kubectl get' as remediation steps.
2. EVERY step must CHANGE cluster state:
   - ImagePullBackOff → kubectl delete pod <name> -n <ns> --grace-period=0 --force
   - CrashLoopBackOff → kubectl rollout restart deployment/<name> -n <ns> OR kubectl delete pod <name> -n <ns>
   - OOMKilled → kubectl delete pod <name> -n <ns> (pod will restart with same limits)
   - Pending → identify the blocking resource and suggest freeing it
3. Use actual pod names from the triage data.
4. Order: safe (restart) before destructive (delete/force).

Output a structured plan with exact kubectl commands.`, triage, metrics, logs, infra)
}

func buildCommanderPrompt(triage, metrics, logs, infra, runbook string, ec *incident.EvidenceCollector, timeline *incident.TimelineEngine) string {
	evidenceText := "(No telemetry evidence recorded)"
	if ec != nil {
		evidenceText = ec.FormatForPrompt()
	}

	timelineText := "(No chronological timeline events recorded)"
	if timeline != nil {
		timelineText = timeline.FormatForAI(25)
	}

	return fmt.Sprintf(`You are the Incident Commander. Synthesize all findings into a final report.
Output ONLY valid JSON — no markdown, no code fences.

INCIDENT TIMELINE:
%s

COLLECTED EVIDENCE:
%s

TRIAGE: %s
METRICS: %s
LOGS: %s
INFRA: %s
RUNBOOK: %s

CRITICAL RULES FOR EVIDENCE-BACKED ROOT CAUSE ANALYSIS (RCA):
1. Every conclusion in "conclusions" MUST be backed by specific evidence ID(s) from COLLECTED EVIDENCE (e.g. ["E1", "E2"]).
2. Use the INCIDENT TIMELINE to establish causality (what occurred first vs cascading symptoms).
3. Distinguish evidence from reasoning:
   - "statement": the factual deduction (e.g. "Redis has no available endpoints")
   - "evidence_ids": valid evidence IDs from COLLECTED EVIDENCE proving this statement
   - "reasoning": how the evidence leads to this conclusion
4. The AI MUST NOT present unsupported conclusions as facts. Any unverified hypothesis or speculation MUST be placed under "hypotheses", NOT in "conclusions".
5. In "root_cause", provide a concise explanation citing the key evidence IDs (e.g. "Redis has no available endpoints [E1, E2]").
6. If the cluster is healthy, set has_issue=false and severity=SEV4.

CRITICAL RULES for remediation_plan:
1. Commands must CHANGE state — never 'kubectl describe' or 'kubectl get'.
2. Use real pod names from triage data with correct -n <namespace>.
3. For ImagePullBackOff: kubectl delete pod <name> -n <ns> --grace-period=0 --force
4. For CrashLoopBackOff: kubectl rollout restart deployment/<name> -n <ns>
5. For OOMKilled: kubectl delete pod <name> -n <ns>

Output EXACTLY this JSON:
{
  "has_issue": true,
  "severity": "SEV1|SEV2|SEV3|SEV4",
  "title": "concise incident title",
  "category": "crashloop|oom|image_pull|pending|node_not_ready|healthy",
  "affected_namespaces": ["ns1"],
  "affected_services": [
    {"service_name": "pod-name", "namespace": "ns", "impact_level": "down|degraded|at_risk"}
  ],
  "root_cause": "concise explanation citing evidence IDs",
  "conclusions": [
    {
      "statement": "Redis has no available endpoints",
      "evidence_ids": ["E1"],
      "reasoning": "Endpoint list is empty while consumer logs report connection refused"
    }
  ],
  "hypotheses": [
    {
      "statement": "Potential upstream network partition",
      "reason": "Inter-node latency telemetry not observable directly",
      "validation_steps": ["ping target node"]
    }
  ],
  "contributing_factors": ["factor1", "factor2"],
  "confidence_score": 85,
  "remediation_plan": [
    {"step_number": 1, "description": "...", "command": "kubectl ...", "is_destructive": true, "is_automated": false}
  ],
  "runbook_used": "crash_loop"
}

If cluster is healthy, set has_issue=false and severity=SEV4.`,
		truncate(timelineText, 2500), truncate(evidenceText, 2500), triage, metrics, logs, infra, runbook)
}

// ─── Parser ───────────────────────────────────────────────────────────────────

type agentOutput struct {
	HasIssue           bool     `json:"has_issue"`
	Severity           string   `json:"severity"`
	Title              string   `json:"title"`
	Category           string   `json:"category"`
	AffectedNamespaces []string `json:"affected_namespaces"`
	AffectedServices   []struct {
		ServiceName string `json:"service_name"`
		Namespace   string `json:"namespace"`
		ImpactLevel string `json:"impact_level"`
	} `json:"affected_services"`
	RootCause           string   `json:"root_cause"`
	ContributingFactors []string `json:"contributing_factors"`
	ConfidenceScore     int      `json:"confidence_score"`
	Conclusions         []struct {
		Statement   string   `json:"statement"`
		EvidenceIDs []string `json:"evidence_ids"`
		Reasoning   string   `json:"reasoning"`
	} `json:"conclusions"`
	Hypotheses []struct {
		Statement       string   `json:"statement"`
		Reason          string   `json:"reason"`
		ValidationSteps []string `json:"validation_steps"`
	} `json:"hypotheses"`
	RemediationPlan []struct {
		StepNumber  int    `json:"step_number"`
		Description string `json:"description"`
		Command     string `json:"command"`
		Destructive bool   `json:"is_destructive"`
		Automated   bool   `json:"is_automated"`
	} `json:"remediation_plan"`
	RunbookUsed string `json:"runbook_used"`
}

func parseReport(raw, clusterName, modelName string, ec *incident.EvidenceCollector, timeline *incident.TimelineEngine) (*incident.Report, error) {
	cleaned := stripMarkdownFences(raw)
	match := jsonBlockRe.FindString(cleaned)
	if match == "" {
		match = cleaned
	}

	var ao agentOutput
	if err := json.Unmarshal([]byte(match), &ao); err != nil {
		r := incident.NewReport(clusterName)
		r.Severity = incident.SEV3
		r.Title = "Pipeline parse error — manual review needed"
		r.HasIssue = true
		r.RootCause = truncate(raw, 500)
		r.ConfidenceScore = 10
		r.LLMModel = modelName
		return r, nil
	}

	r := incident.NewReport(clusterName)
	r.HasIssue = ao.HasIssue
	r.Severity = incident.Severity(ao.Severity)
	r.Title = ao.Title
	r.Category = ao.Category
	r.AffectedNamespaces = ao.AffectedNamespaces
	r.RootCause = ao.RootCause
	r.ContributingFactors = ao.ContributingFactors
	r.ConfidenceScore = ao.ConfidenceScore
	r.RunbookUsed = ao.RunbookUsed
	r.LLMModel = modelName

	for _, svc := range ao.AffectedServices {
		r.AffectedServices = append(r.AffectedServices, incident.AffectedService{
			ServiceName: svc.ServiceName,
			Namespace:   svc.Namespace,
			ImpactLevel: svc.ImpactLevel,
		})
	}
	for _, step := range ao.RemediationPlan {
		r.RemediationPlan = append(r.RemediationPlan, incident.RemediationStep{
			StepNumber:  step.StepNumber,
			Description: step.Description,
			Command:     step.Command,
			Destructive: step.Destructive,
			Automated:   step.Automated,
		})
	}

	// ─── Process Evidence-Backed Conclusions & Guardrails ───────────────────────
	seenEvidence := make(map[string]bool)

	for _, c := range ao.Conclusions {
		var validIDs []string
		for _, id := range c.EvidenceIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				validIDs = append(validIDs, id)
			}
		}

		if len(validIDs) == 0 {
			r.Hypotheses = append(r.Hypotheses, incident.Hypothesis{
				Statement: c.Statement,
				Reason:    "Demoted from conclusion: unsupported by direct telemetry evidence IDs",
			})
			continue
		}

		if ec != nil && len(ec.All()) > 0 {
			var verifiedIDs []string
			for _, id := range validIDs {
				if ev, ok := ec.GetByID(id); ok {
					verifiedIDs = append(verifiedIDs, id)
					if !seenEvidence[id] {
						seenEvidence[id] = true
						r.Evidence = append(r.Evidence, ev)
					}
				}
			}

			if len(verifiedIDs) == 0 {
				r.Hypotheses = append(r.Hypotheses, incident.Hypothesis{
					Statement: c.Statement,
					Reason:    fmt.Sprintf("Demoted from conclusion: cited evidence IDs (%s) not found in collected telemetry", strings.Join(validIDs, ", ")),
				})
				continue
			}
			validIDs = verifiedIDs
		}

		r.Conclusions = append(r.Conclusions, incident.Conclusion{
			Statement:   c.Statement,
			EvidenceIDs: validIDs,
			Reasoning:   c.Reasoning,
		})
	}

	for _, h := range ao.Hypotheses {
		r.Hypotheses = append(r.Hypotheses, incident.Hypothesis{
			Statement:       h.Statement,
			Reason:          h.Reason,
			ValidationSteps: h.ValidationSteps,
		})
	}

	if len(r.Evidence) == 0 && ec != nil {
		allEv := ec.All()
		if len(allEv) > 10 {
			r.Evidence = allEv[:10]
		} else {
			r.Evidence = allEv
		}
	}

	if timeline != nil {
		r.Timeline = timeline.Events()
	}

	if r.RootCause == "" && len(r.Conclusions) > 0 {
		c := r.Conclusions[0]
		if len(c.EvidenceIDs) > 0 {
			r.RootCause = fmt.Sprintf("%s [%s]", c.Statement, strings.Join(c.EvidenceIDs, ", "))
		} else {
			r.RootCause = c.Statement
		}
	}

	return r, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func fullModelName(cfg *config.Config) string {
	return cfg.LLMProvider + "/" + cfg.LLMModel
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func isRateLimit(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") || strings.Contains(msg, "rate limit")
}

func stripMarkdownFences(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
