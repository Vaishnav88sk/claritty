// Package ai implements the LLM-powered SRE agent pipeline using langchaingo.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/k8s"
)

const maxRetries = 3

var retryBackoffs = []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second}

// Pipeline orchestrates the multi-agent SRE scan workflow.
type Pipeline struct {
	cfg    *config.Config
	k8sCli *k8s.Client
	llm    llms.Model
}

// New creates a new Pipeline with a configured LLM and Kubernetes client.
func New(cfg *config.Config, k8sCli *k8s.Client) (*Pipeline, error) {
	llm, err := buildLLM(cfg)
	if err != nil {
		return nil, fmt.Errorf("init LLM: %w", err)
	}
	return &Pipeline{cfg: cfg, k8sCli: k8sCli, llm: llm}, nil
}

// buildLLM creates the appropriate LLM client based on config.
func buildLLM(cfg *config.Config) (llms.Model, error) {
	modelName := cfg.LLMModel
	switch cfg.LLMProvider {
	case "groq":
		modelName = strings.TrimPrefix(modelName, "groq/")
		modelName = strings.TrimPrefix(modelName, "groq/") // Handle double prefix
		return openai.New(
			openai.WithBaseURL("https://api.groq.com/openai/v1"),
			openai.WithToken(cfg.GroqAPIKey),
			openai.WithModel(modelName),
		)
	case "openai":
		return openai.New(
			openai.WithToken(cfg.OpenAIAPIKey),
			openai.WithModel(modelName),
		)
	case "mistral":
		modelName = strings.TrimPrefix(modelName, "mistral/")
		modelName = strings.TrimPrefix(modelName, "mistral/")
		return openai.New(
			openai.WithBaseURL("https://api.mistral.ai/v1"),
			openai.WithToken(cfg.MistralAPIKey),
			openai.WithModel(modelName),
		)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLMProvider)
	}
}

// RunScan executes the full 6-stage SRE pipeline and returns a structured report.
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
			fmt.Printf("\n⚠  Rate limit hit — retrying in %s (%d/%d)...\n\n", wait, attempt+2, maxRetries)
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

// runPipeline executes the sequential 6-stage agent pipeline.
func (p *Pipeline) runPipeline(ctx context.Context) (*incident.Report, error) {
	// ── Stage 1: Triage ────────────────────────────────────────────────────
	fmt.Println("  🔍 Stage 1/6: Triage Agent — scanning cluster...")
	triageData, err := p.collectTriageData(ctx)
	if err != nil {
		return nil, fmt.Errorf("triage data collection: %w", err)
	}
	triageSummary, err := p.callAgent(ctx, buildTriagePrompt(triageData, p.cfg))
	if err != nil {
		return nil, fmt.Errorf("triage agent: %w", err)
	}

	// ── Stage 2: Metrics ───────────────────────────────────────────────────
	fmt.Println("  📈 Stage 2/6: Metrics Agent — analyzing telemetry...")
	metricsSummary, err := p.callAgent(ctx, buildMetricsPrompt(truncate(triageSummary, 2000), p.cfg))
	if err != nil {
		return nil, fmt.Errorf("metrics agent: %w", err)
	}

	// ── Stage 3: Logs ──────────────────────────────────────────────────────
	fmt.Println("  📋 Stage 3/6: Log Agent — mining error patterns...")
	logData := p.collectLogData(ctx, truncate(triageSummary, 1000))
	logSummary, err := p.callAgent(ctx, buildLogPrompt(
		truncate(triageSummary, 1000),
		truncate(metricsSummary, 1000),
		logData,
		p.cfg,
	))
	if err != nil {
		return nil, fmt.Errorf("log agent: %w", err)
	}

	// ── Stage 4: Infrastructure ────────────────────────────────────────────
	fmt.Println("  🏗  Stage 4/6: Infra Agent — diagnosing K8s constraints...")
	infraSummary, err := p.callAgent(ctx, buildInfraPrompt(
		truncate(triageSummary, 1000),
		truncate(metricsSummary, 1000),
		truncate(logSummary, 1000),
	))
	if err != nil {
		return nil, fmt.Errorf("infra agent: %w", err)
	}

	// ── Stage 5: Runbook ───────────────────────────────────────────────────
	fmt.Println("  📖 Stage 5/6: Runbook Agent — selecting remediation plan...")
	runbookSummary, err := p.callAgent(ctx, buildRunbookPrompt(
		truncate(triageSummary, 800),
		truncate(metricsSummary, 800),
		truncate(logSummary, 800),
		truncate(infraSummary, 800),
		p.cfg,
	))
	if err != nil {
		return nil, fmt.Errorf("runbook agent: %w", err)
	}

	// ── Stage 6: Commander ─────────────────────────────────────────────────
	fmt.Println("  ⚡ Stage 6/6: Incident Commander — synthesizing final report...")
	finalJSON, err := p.callAgent(ctx, buildCommanderPrompt(
		truncate(triageSummary, 800),
		truncate(metricsSummary, 800),
		truncate(logSummary, 800),
		truncate(infraSummary, 800),
		truncate(runbookSummary, 1000),
	))
	if err != nil {
		return nil, fmt.Errorf("commander agent: %w", err)
	}

	return parseReport(finalJSON, p.cfg.LLMModel)
}

// callAgent sends a prompt to the LLM and returns the text response.
func (p *Pipeline) callAgent(ctx context.Context, prompt string) (string, error) {
	resp, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt,
		llms.WithTemperature(p.cfg.LLMTemperature),
		llms.WithMaxTokens(p.cfg.LLMMaxTokens),
	)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// ─── Data Collection ──────────────────────────────────────────────────────────

type triageCollectedData struct {
	Pods      []k8s.PodSummary
	Events    []k8s.EventSummary
	Nodes     []k8s.NodeSummary
	Summaries []k8s.NamespaceSummary
}

func (p *Pipeline) collectTriageData(ctx context.Context) (*triageCollectedData, error) {
	allPods, err := p.k8sCli.ListPods(ctx, p.cfg.Namespaces)
	if err != nil {
		return nil, err
	}

	// Filter for problematic pods to reduce payload size (prevents 413 error)
	var problemPods []k8s.PodSummary
	for _, pod := range allPods {
		isHealthy := pod.Phase == "Running" && pod.Ready && pod.Restarts <= int32(p.cfg.RestartWarningCnt)
		if !isHealthy {
			problemPods = append(problemPods, pod)
		}
	}

	// If we have too many problem pods, limit to top 50 to stay within token/payload limits
	if len(problemPods) > 50 {
		problemPods = problemPods[:50]
	}

	events, _ := p.k8sCli.GetWarningEvents(ctx, p.cfg.Namespaces, 20) // Limit to 20 most recent warnings
	nodes, _ := p.k8sCli.GetNodeHealth(ctx)
	summaries, _ := p.k8sCli.GetNamespaceSummaries(ctx, p.cfg.Namespaces)

	return &triageCollectedData{
		Pods: problemPods, Events: events, Nodes: nodes, Summaries: summaries,
	}, nil
}

func (p *Pipeline) collectLogData(ctx context.Context, triage string) string {
	// Parse triage for problem pod names (best-effort)
	var logs strings.Builder
	for _, pod := range extractMentionedPods(triage) {
		for _, ns := range p.cfg.Namespaces {
			log, err := p.k8sCli.GetPodLogs(ctx, ns, pod, "", int64(p.cfg.PodLogTailLines), false)
			if err == nil && log != "" {
				fmt.Fprintf(&logs, "=== Logs for %s/%s ===\n%s\n", ns, pod, truncate(log, 2000))
			}
			// Also fetch previous container logs for crash loop detection
			prevLog, err := p.k8sCli.GetPodLogs(ctx, ns, pod, "", int64(p.cfg.PodLogTailLines), true)
			if err == nil && prevLog != "" {
				fmt.Fprintf(&logs, "=== PREVIOUS Logs for %s/%s ===\n%s\n", ns, pod, truncate(prevLog, 1000))
			}
		}
	}
	return logs.String()
}

// ─── Prompt builders ─────────────────────────────────────────────────────────

func buildTriagePrompt(data *triageCollectedData, cfg *config.Config) string {
	podJSON, _ := json.Marshal(data.Pods)
	eventJSON, _ := json.Marshal(data.Events)
	nodeJSON, _ := json.Marshal(data.Nodes)
	summJSON, _ := json.Marshal(data.Summaries)

	return fmt.Sprintf(`You are an SRE Triage Specialist with 10 years of experience.
Analyze the following Kubernetes cluster data and produce a comprehensive triage report.

PODS:
%s

WARNING EVENTS:
%s

NODE HEALTH:
%s

NAMESPACE SUMMARY:
%s

Namespaces being monitored: %s

Instructions:
1. Identify all pods NOT in Running phase or with high restart counts (>%d).
2. List all Warning events and their severity.
3. Assess node health — are any nodes Not Ready or cordoned?
4. Classify severity: SEV1 (service down), SEV2 (major degradation), SEV3 (minor), SEV4 (healthy).
5. Output a structured triage report identifying the scope and initial severity.`,
		truncate(string(podJSON), 4000),
		truncate(string(eventJSON), 2000),
		truncate(string(nodeJSON), 1000),
		truncate(string(summJSON), 1000),
		strings.Join(cfg.Namespaces, ", "),
		cfg.RestartWarningCnt,
	)
}

func buildMetricsPrompt(triage string, cfg *config.Config) string {
	return fmt.Sprintf(`You are a Metrics & Telemetry Analyst.
Based on the triage below, analyze what metrics data would confirm the root cause.

TRIAGE REPORT:
%s

Note: Prometheus URL is %s (enabled: %v). 
CPU critical threshold: %.0f%%, Memory critical threshold: %.0f%%, Restart threshold: %d.

Since you cannot directly query Prometheus from this context, analyze the triage data to infer:
1. Which pods likely have high CPU or memory pressure based on OOMKilled/CrashLoop events.
2. Estimated restart rates based on restart counts.
3. Likely resource bottlenecks.
4. Whether this appears to be resource exhaustion vs application error.

Output a structured metrics analysis report.`,
		triage,
		cfg.PrometheusURL, cfg.PrometheusEnabled,
		cfg.CPUCriticalPct, cfg.MemCriticalPct, cfg.RestartCriticalCnt,
	)
}

func buildLogPrompt(triage, metrics, logData string, cfg *config.Config) string {
	if logData == "" {
		logData = "(No logs collected — no problem pods identified or pods not accessible)"
	}
	return fmt.Sprintf(`You are a Log Analysis & Pattern Mining Agent.
Analyze the following pod logs to identify root cause error patterns.

TRIAGE:
%s

METRICS:
%s

ACTUAL POD LOGS:
%s

Instructions:
1. Identify the most critical error messages (exceptions, panics, OOM, connection errors).
2. Classify errors: application bug / config error / resource exhaustion / network issue.
3. Find the first occurrence timestamp of errors (when did this start?).
4. For CrashLoopBackOff: focus on PREVIOUS container logs.
5. Extract up to 5 most relevant log lines that reveal the root cause.

Output a structured log analysis report.`,
		truncate(triage, 1500), truncate(metrics, 1000), truncate(logData, 3000),
	)
}

func buildInfraPrompt(triage, metrics, logs string) string {
	return fmt.Sprintf(`You are a Kubernetes Infrastructure Diagnostician.
Based on the findings below, diagnose if infrastructure constraints are causing the incident.

TRIAGE: %s

METRICS: %s

LOGS: %s

Analyze:
1. Are resource quotas being hit in any namespace?
2. Are there PVC binding issues?
3. Are HPAs at max replicas and unable to scale?
4. Are node taints blocking scheduling?
5. Are init containers failing?
6. Is this an infrastructure cause (quotas, storage, scheduling) or application cause (code bug, config)?

Output a structured infrastructure diagnosis.`,
		truncate(triage, 1200), truncate(metrics, 800), truncate(logs, 800),
	)
}

func buildRunbookPrompt(triage, metrics, logs, infra string, cfg *config.Config) string {
	dryRunMode := "ENABLED (suggest commands only — do NOT execute)"
	if !cfg.DryRun {
		dryRunMode = "DISABLED (commands will be executed)"
	}
	return fmt.Sprintf(`You are a senior Runbook & Remediation Engineer with 10+ years of Kubernetes experience.
Your job is to create an ACTIONABLE remediation plan that FIXES the problem — not just inspects it.

TRIAGE: %s
METRICS: %s
LOGS: %s
INFRA: %s

DRY RUN MODE: %s

CRITICAL RULES:
1. DO NOT suggest 'kubectl describe' or 'kubectl get' as remediation steps. Those are for humans to inspect manually — NOT for automated remediation.
2. EVERY step must be a command that CHANGES cluster state or FIXES the problem:
   - ImagePullBackOff → kubectl patch pod <name> -n <ns> -p '{"spec":{...}}' with corrected image, or kubectl delete pod <name> -n <ns>
   - CrashLoopBackOff → kubectl rollout restart deployment/<name> -n <ns>, or kubectl delete pod <name> -n <ns> --grace-period=0
   - OOMKilled → kubectl patch deployment <name> -n <ns> --type='json' -p='[{"op":"replace","path":"/spec/template/spec/containers/0/resources/limits/memory","value":"512Mi"}]'
   - Pending (resource) → kubectl describe the quota (OK for diagnosis only), then suggest kubectl delete pod or scale down other pods
   - Node NotReady → kubectl drain <node> --ignore-daemonsets, or kubectl uncordon <node>
3. Order steps: safe non-destructive first (restart) → destructive last (delete/force)
4. If a pod name is known, include it in the command.
5. Use correct namespaces in all commands.

Output a structured remediation plan with exact executable kubectl commands.`,
		truncate(triage, 800), truncate(metrics, 600), truncate(logs, 600), truncate(infra, 600),
		dryRunMode,
	)
}

func buildCommanderPrompt(triage, metrics, logs, infra, runbook string) string {
	return fmt.Sprintf(`You are the Incident Commander. Synthesize all findings into a final incident report.
Output ONLY valid JSON — no markdown, no explanation, no code fences.

TRIAGE: %s
METRICS: %s
LOGS: %s
INFRA: %s
RUNBOOK PLAN: %s

CRITICAL RULES FOR remediation_plan:
1. Each command MUST be an executable kubectl command that CHANGES state (fixes the problem).
2. NEVER put 'kubectl describe', 'kubectl get', or 'kubectl logs' as a remediation step — those are diagnostic, not remediation.
3. Use the ACTUAL pod/deployment names found in the triage data.
4. Use the correct namespace in every command (use -n <namespace>).
5. For ImagePullBackOff: delete the pod so it can be recreated → 'kubectl delete pod <name> -n <ns> --grace-period=0 --force'
6. For CrashLoopBackOff: restart the deployment or delete the pod → 'kubectl rollout restart deployment/<name> -n <ns>'
7. For OOMKilled: patch memory limits or delete the pod → 'kubectl delete pod <name> -n <ns>'
8. For Pending (no resources): identify what is blocking and suggest removing/scaling the blocker.

Output EXACTLY this JSON structure:
{
  "has_issue": true,
  "severity": "SEV1|SEV2|SEV3|SEV4",
  "title": "concise incident title",
  "category": "crashloop|oom|high_cpu|high_memory|image_pull|pending|node_not_ready|disk_pressure|error_rate|latency|healthy",
  "affected_namespaces": ["ns1"],
  "affected_services": [{"service_name": "x", "namespace": "y", "impact_level": "down|degraded|at_risk"}],
  "root_cause": "3-5 sentence explanation of the actual root cause",
  "contributing_factors": ["factor1", "factor2"],
  "confidence_score": 85,
  "remediation_plan": [
    {"step_number": 1, "description": "Force delete the crashing pod so Kubernetes recreates it cleanly", "command": "kubectl delete pod crashloop-demo -n default --grace-period=0 --force", "is_destructive": true, "is_automated": true},
    {"step_number": 2, "description": "Fix the invalid image tag and delete the stuck pod", "command": "kubectl delete pod badimage-demo -n default --grace-period=0 --force", "is_destructive": true, "is_automated": true}
  ],
  "runbook_used": "crash_loop.yaml"
}

Only report CONFIRMED issues with real affected pod names from the triage data. If cluster is healthy, set has_issue=false and severity=SEV4.`,
		truncate(triage, 1000), truncate(metrics, 700),
		truncate(logs, 700), truncate(infra, 700), truncate(runbook, 800),
	)
}

// ─── Report Parser ────────────────────────────────────────────────────────────

// agentOutput is the raw JSON structure from the commander agent.
type agentOutput struct {
	HasIssue           bool   `json:"has_issue"`
	Severity           string `json:"severity"`
	Title              string `json:"title"`
	Category           string `json:"category"`
	AffectedNamespaces []string `json:"affected_namespaces"`
	AffectedServices   []struct {
		ServiceName string `json:"service_name"`
		Namespace   string `json:"namespace"`
		ImpactLevel string `json:"impact_level"`
	} `json:"affected_services"`
	RootCause          string   `json:"root_cause"`
	ContributingFactors []string `json:"contributing_factors"`
	ConfidenceScore    int      `json:"confidence_score"`
	RemediationPlan    []struct {
		StepNumber  int    `json:"step_number"`
		Description string `json:"description"`
		Command     string `json:"command"`
		Destructive bool   `json:"is_destructive"`
		Automated   bool   `json:"is_automated"`
	} `json:"remediation_plan"`
	RunbookUsed string `json:"runbook_used"`
}

var jsonBlockRe = regexp.MustCompile(`(?s)\{.*\}`)

func parseReport(raw, llmModel string) (*incident.Report, error) {
	cleaned := stripMarkdownFences(raw)
	match := jsonBlockRe.FindString(cleaned)
	if match == "" {
		match = cleaned
	}

	var ao agentOutput
	if err := json.Unmarshal([]byte(match), &ao); err != nil {
		// Build a minimal error report so the CLI doesn't crash
		r := incident.NewReport()
		r.Severity = incident.SEV3
		r.Title = "Pipeline parse error — manual review needed"
		r.Category = "unknown"
		r.HasIssue = true
		r.RootCause = fmt.Sprintf("Agent output could not be parsed: %s", truncate(raw, 500))
		r.ConfidenceScore = 10
		r.LLMModel = llmModel
		r.RawAgentOutput = truncate(raw, 2000)
		return r, nil
	}

	if !ao.HasIssue {
		r := incident.NewReport()
		r.Severity = incident.SEV4
		r.Title = "Cluster Healthy — No Issues Detected"
		r.Category = "healthy"
		r.HasIssue = false
		r.Status = incident.StatusResolved
		r.RootCause = "All monitored systems are operating within normal parameters."
		r.ConfidenceScore = ao.ConfidenceScore
		r.LLMModel = llmModel
		return r, nil
	}

	r := incident.NewReport()
	r.Severity = parseSeverity(ao.Severity)
	r.Title = ao.Title
	r.Category = ao.Category
	r.HasIssue = true
	r.Status = incident.StatusInvestigating
	r.AffectedNamespaces = ao.AffectedNamespaces
	r.RootCause = ao.RootCause
	r.ContributingFactors = ao.ContributingFactors
	r.ConfidenceScore = ao.ConfidenceScore
	r.RunbookUsed = ao.RunbookUsed
	r.LLMModel = llmModel
	r.RawAgentOutput = truncate(raw, 2000)

	for _, svc := range ao.AffectedServices {
		r.AffectedServices = append(r.AffectedServices, incident.ServiceImpact{
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
			Status:      "PENDING",
		})
	}
	return r, nil
}

func parseSeverity(s string) incident.Severity {
	switch strings.ToUpper(s) {
	case "SEV1":
		return incident.SEV1
	case "SEV2":
		return incident.SEV2
	case "SEV3":
		return incident.SEV3
	default:
		return incident.SEV4
	}
}

func stripMarkdownFences(s string) string {
	s = regexp.MustCompile("```(?:json)?").ReplaceAllString(s, "")
	return strings.ReplaceAll(s, "```", "")
}

func isRateLimit(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "ratelimit") ||
		strings.Contains(msg, "quota exceeded")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// extractMentionedPods does a simple regex-based extraction of pod names from triage text.
func extractMentionedPods(text string) []string {
	re := regexp.MustCompile(`\b([a-z0-9][a-z0-9\-]{2,62})\b`)
	matches := re.FindAllString(text, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		if len(m) > 8 && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	if len(out) > 5 {
		return out[:5]
	}
	return out
}
