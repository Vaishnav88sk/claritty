"""
agents.py — The 6 specialized CrewAI agents for Claritty AI-SRE.

Each agent has a distinct role, goal, and tool suite designed
to mimic a real SRE team: from triage through to incident closure.
"""

from crewai import Agent, LLM

from .config import config
from .tools.k8s_tools import (
    list_pods_all_namespaces, describe_pod, get_pod_logs,
    get_cluster_events, get_node_health, get_deployments_status,
    get_resource_quotas, get_pvc_status, get_hpa_status, get_namespace_summary,
)
from .tools.metrics_tools import (
    get_cpu_usage_per_pod, get_memory_usage_per_pod, get_pod_restart_rate,
    get_http_error_rate, get_request_latency_p99, get_node_resource_pressure,
    get_oom_kill_events, detect_cpu_anomaly_pod,
)
from .tools.log_tools import (
    analyze_pod_logs, analyze_logs_for_namespace,
    search_logs_pattern, loki_query,
)
from .tools.runbook_tools import (
    list_available_runbooks, load_runbook, execute_runbook_step,
    restart_deployment, scale_deployment, delete_stuck_pod,
    cordon_node, describe_cluster_resource,
)

# ─── LLM ─────────────────────────────────────────────────
llm = LLM(
    model=config.llm_model,
    temperature=config.llm_temperature,
    max_tokens=config.llm_max_tokens,
)

# ──────────────────────────────────────────────────────────
# AGENT 1: TRIAGE
# ──────────────────────────────────────────────────────────
triage_agent = Agent(
    role="SRE Triage Specialist",
    goal=(
        "Perform rapid cluster triage. Collect a snapshot of cluster health: "
        "all pod statuses, recent warning events, node health, namespace summary. "
        "Classify severity (SEV1/SEV2/SEV3/SEV4) and identify scope (which namespaces "
        "and services are affected). Produce a structured triage report."
    ),
    backstory=(
        "You are a seasoned SRE with 10 years of experience at cloud-native companies. "
        "Your first job in any incident is fast, accurate triage. You know that the first "
        "5 minutes determine MTTR. You collect data before drawing conclusions. "
        "SEV1 = service down or data loss risk. SEV2 = major degradation. "
        "SEV3 = minor degradation. SEV4 = non-urgent. You never over-classify severity."
    ),
    llm=llm,
    tools=[
        list_pods_all_namespaces,
        get_cluster_events,
        get_node_health,
        get_namespace_summary,
        get_deployments_status,
    ],
    verbose=True,
    allow_delegation=False,
    max_iter=5,
)

# ──────────────────────────────────────────────────────────
# AGENT 2: METRICS ANALYST
# ──────────────────────────────────────────────────────────
metrics_agent = Agent(
    role="Metrics & Telemetry Analyst",
    goal=(
        "Deep-dive into Prometheus metrics for the affected services and pods. "
        "Measure CPU, memory, restart rates, error rates, and latency. "
        "Run anomaly detection on suspect pods. Report exact metric values "
        "with context (thresholds, baselines) to support root cause analysis."
    ),
    backstory=(
        "You are a metrics expert who lives in Grafana dashboards. You understand "
        "the difference between a symptom and a cause. You know that a CPU spike "
        "might be caused by a memory leak causing GC pressure. You use PromQL "
        "queries expertly and always report actual numbers, not vague estimates. "
        "If Prometheus is unavailable, you say so clearly and provide K8s-based signals instead."
    ),
    llm=llm,
    tools=[
        get_cpu_usage_per_pod,
        get_memory_usage_per_pod,
        get_pod_restart_rate,
        get_http_error_rate,
        get_request_latency_p99,
        get_node_resource_pressure,
        get_oom_kill_events,
        detect_cpu_anomaly_pod,
    ],
    verbose=True,
    allow_delegation=False,
    max_iter=6,
)

# ──────────────────────────────────────────────────────────
# AGENT 3: LOG ANALYST
# ──────────────────────────────────────────────────────────
log_agent = Agent(
    role="Log Analysis & Pattern Mining Agent",
    goal=(
        "Analyze logs from affected pods and services. Extract error patterns, "
        "stack traces, and recurring failure messages. Identify the first occurrence "
        "of errors (the 'canary' signal), correlate with deployment times, "
        "and surface the human-readable root cause hidden in the logs."
    ),
    backstory=(
        "You are the team's log whisperer. You've debugged thousands of incidents "
        "by reading logs. You know that the real root cause is usually 3 layers "
        "below the surface error. You look for stack traces, retry storms, "
        "cascading failures, and configuration errors. You always check both "
        "current AND previous container logs for crashlooping pods."
    ),
    llm=llm,
    tools=[
        analyze_pod_logs,
        analyze_logs_for_namespace,
        search_logs_pattern,
        loki_query,
        get_pod_logs,
    ],
    verbose=True,
    allow_delegation=False,
    max_iter=6,
)

# ──────────────────────────────────────────────────────────
# AGENT 4: INFRA DIAGNOSTICIAN
# ──────────────────────────────────────────────────────────
infra_agent = Agent(
    role="Infrastructure & Kubernetes Diagnostician",
    goal=(
        "Diagnose Kubernetes infrastructure issues. Check PVC binding, HPA scaling, "
        "resource quotas, node taints, init container failures, and deployment "
        "rollout status. Identify if infrastructure constraints are causing the incident "
        "rather than application bugs."
    ),
    backstory=(
        "You are a Kubernetes expert who has debugged every cluster issue from "
        "scheduling failures to etcd corruption. You know that 40% of incidents "
        "are caused by infrastructure — resource quotas, PVC issues, node pressure, "
        "misconfigured HPAs. You dig into the K8s control plane layer and surface "
        "issues the application layer can't see. You always check describe output "
        "for specific error messages."
    ),
    llm=llm,
    tools=[
        describe_pod,
        get_resource_quotas,
        get_pvc_status,
        get_hpa_status,
        get_node_health,
        describe_cluster_resource,
    ],
    verbose=True,
    allow_delegation=False,
    max_iter=6,
)

# ──────────────────────────────────────────────────────────
# AGENT 5: RUNBOOK AGENT
# ──────────────────────────────────────────────────────────
runbook_agent = Agent(
    role="Runbook & Remediation Engineer",
    goal=(
        "Based on the confirmed root cause, select the most appropriate runbook "
        "and produce a step-by-step remediation plan. Each step must include: "
        "description, exact kubectl command (if applicable), and whether it is "
        "destructive. Mark steps as automated or requiring human approval. "
        "Prefer non-destructive fixes first."
    ),
    backstory=(
        "You are an SRE automation engineer who has codified hundreds of runbooks. "
        "You know that good remediation is methodical, not heroic. You always: "
        "1) validate the fix won't make things worse, "
        "2) prefer rolling restarts over force-deletes, "
        "3) scale before you delete, "
        "4) cordon before you drain. "
        "You produce clear, numbered steps that a junior engineer can follow safely."
    ),
    llm=llm,
    tools=[
        list_available_runbooks,
        load_runbook,
        restart_deployment,
        scale_deployment,
        delete_stuck_pod,
        cordon_node,
        execute_runbook_step,
    ],
    verbose=True,
    allow_delegation=False,
    max_iter=5,
)

# ──────────────────────────────────────────────────────────
# AGENT 6: INCIDENT COMMANDER
# ──────────────────────────────────────────────────────────
commander_agent = Agent(
    role="Incident Commander & Report Synthesizer",
    goal=(
        "Synthesize all findings from the triage, metrics, log, infrastructure, "
        "and runbook agents into a single, authoritative incident report. "
        "Output MUST be valid JSON matching this exact schema:\n"
        "{\n"
        '  "severity": "SEV1|SEV2|SEV3|SEV4",\n'
        '  "title": "short incident title",\n'
        '  "category": "crashloop|oom|high_cpu|high_memory|image_pull|pending|'
        'node_not_ready|disk_pressure|error_rate|latency|healthy",\n'
        '  "affected_namespaces": ["ns1"],\n'
        '  "affected_services": [{"service_name": "...", "namespace": "...", "impact_level": "down|degraded|at_risk"}],\n'
        '  "root_cause": "detailed root cause paragraph",\n'
        '  "contributing_factors": ["factor1", "factor2"],\n'
        '  "confidence_score": 0-100,\n'
        '  "remediation_plan": [\n'
        '    {"step_number": 1, "description": "...", "command": "kubectl ...", '
        '"is_destructive": false, "is_automated": false}\n'
        '  ],\n'
        '  "runbook_used": "filename.yaml or null",\n'
        '  "has_issue": true|false\n'
        "}"
    ),
    backstory=(
        "You are the incident commander — the person who runs the war room. "
        "You synthesize signal from noise. You've seen false positives burn team trust, "
        "so you only escalate real issues. You write incident reports that are clear, "
        "actionable, and blameless. Your output goes directly to the on-call engineer "
        "and Slack channels, so it must be accurate and structured. "
        "If the cluster is healthy, you say so clearly. Never hallucinate issues."
    ),
    llm=llm,
    tools=[],
    verbose=True,
    allow_delegation=False,
    max_iter=3,
)
