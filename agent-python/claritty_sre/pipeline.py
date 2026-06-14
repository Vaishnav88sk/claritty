"""
pipeline.py — CrewAI incident pipeline for Claritty AI-SRE.

Orchestrates the 6-agent swarm through a sequential pipeline:
Triage → Metrics → Logs → Infra → Runbook → Commander

Each agent's output feeds into the next as context.
Final output is a structured IncidentReport.
"""

import json
import logging
import re
import time
from datetime import datetime
from typing import Optional

# Max retries for LLM rate limit errors
MAX_RETRIES = 3
RETRY_BACKOFF = [15, 30, 60]  # seconds between retries

from crewai import Crew, Task, Process

from .agents import (
    triage_agent, metrics_agent, log_agent,
    infra_agent, runbook_agent, commander_agent,
)
from .config import config
from .incident import (
    IncidentReport, Severity, IncidentStatus, Evidence, EvidenceType,
    RemediationStep, ServiceImpact, ClusterHealthSnapshot,
)

logger = logging.getLogger("claritty.pipeline")


def _clean_json(raw: str) -> str:
    """Strip markdown code fences from LLM JSON output."""
    return re.sub(r"```(?:json)?|```", "", raw).strip()


def _parse_incident_json(raw: str, llm_model: str, duration: float) -> IncidentReport:
    """Parse the commander's JSON output into a validated IncidentReport."""
    cleaned = _clean_json(raw)

    # Find JSON block if surrounded by text
    match = re.search(r"\{.*\}", cleaned, re.DOTALL)
    if match:
        cleaned = match.group(0)

    try:
        data = json.loads(cleaned)
    except json.JSONDecodeError as e:
        logger.warning("JSON parse failed (%s), building minimal report", e)
        data = {
            "severity": "SEV3",
            "title": "Pipeline parse error — manual review needed",
            "category": "unknown",
            "has_issue": True,
            "root_cause": f"Agent output could not be parsed: {raw[:500]}",
            "confidence_score": 10,
        }

    has_issue = data.get("has_issue", True)

    if not has_issue:
        return IncidentReport(
            severity=Severity.SEV4,
            title="Cluster Healthy — No Issues Detected",
            category="healthy",
            status=IncidentStatus.RESOLVED,
            root_cause="All monitored systems are operating within normal parameters.",
            confidence_score=data.get("confidence_score", 95),
            llm_model=llm_model,
            scan_duration_seconds=duration,
        )

    # Parse severity
    sev_str = data.get("severity", "SEV3").upper()
    try:
        severity = Severity[sev_str]
    except KeyError:
        severity = Severity.SEV3

    # Parse services
    services = []
    for svc in data.get("affected_services", []):
        try:
            services.append(ServiceImpact(
                service_name=svc.get("service_name", "unknown"),
                namespace=svc.get("namespace", "default"),
                impact_level=svc.get("impact_level", "degraded"),
            ))
        except Exception:
            pass

    # Parse remediation steps
    steps = []
    for step in data.get("remediation_plan", []):
        try:
            steps.append(RemediationStep(
                step_number=step.get("step_number", len(steps) + 1),
                description=step.get("description", ""),
                command=step.get("command"),
                is_destructive=step.get("is_destructive", False),
                is_automated=step.get("is_automated", False),
            ))
        except Exception:
            pass

    report = IncidentReport(
        severity=severity,
        title=data.get("title", "Unnamed Incident"),
        category=data.get("category", "unknown"),
        status=IncidentStatus.INVESTIGATING,
        affected_namespaces=data.get("affected_namespaces", []),
        affected_services=services,
        root_cause=data.get("root_cause", ""),
        contributing_factors=data.get("contributing_factors", []),
        confidence_score=data.get("confidence_score", 0),
        remediation_plan=steps,
        runbook_used=data.get("runbook_used"),
        llm_model=llm_model,
        scan_duration_seconds=duration,
        raw_agent_output=raw[:2000],
    )
    return report


def build_crew() -> Crew:
    """Build the 6-task sequential crew pipeline."""
    namespaces_str = ", ".join(config.namespaces)

    task_triage = Task(
        description=(
            f"Perform cluster triage across namespaces: {namespaces_str}.\n"
            "1. List all pods and their phases/restarts\n"
            "2. Get all Warning events from the last hour\n"
            "3. Check node health and conditions\n"
            "4. Get namespace summary\n"
            "5. Classify initial severity (SEV1-SEV4) based on what you find\n"
            "6. Identify which pods/services appear affected\n"
            "Output: a structured triage summary with severity classification and scope."
        ),
        expected_output=(
            "Triage report with: severity classification, affected pods list, "
            "warning events summary, node health status, and initial scope assessment."
        ),
        agent=triage_agent,
    )

    task_metrics = Task(
        description=(
            "Based on the triage findings, deep-dive into metrics.\n"
            "1. Get CPU and memory usage for top pods\n"
            "2. Check pod restart rates (last 30 min)\n"
            "3. Check HTTP error rates and P99 latency\n"
            "4. Check node CPU/memory pressure\n"
            "5. Check for OOM kill events\n"
            "6. If a specific pod is suspect, run anomaly detection on it\n"
            "Report exact metric values with context about whether they exceed thresholds.\n"
            f"CPU critical threshold: {config.cpu_critical_pct}%\n"
            f"Memory critical threshold: {config.memory_critical_pct}%\n"
            f"Restart critical threshold: {config.restart_critical_count}"
        ),
        expected_output=(
            "Metrics analysis with: exact CPU/memory values per pod, restart rates, "
            "error rates, OOM events, and anomaly detection results for suspect pods."
        ),
        agent=metrics_agent,
        context=[task_triage],
    )

    task_logs = Task(
        description=(
            "Analyze logs from the pods identified as problematic in triage and metrics.\n"
            "1. Fetch and analyze logs for the top 5 most troubled pods\n"
            "2. Extract error patterns and classify them\n"
            "3. Pull stack traces if present\n"
            "4. For crashlooping pods, check PREVIOUS container logs\n"
            "5. Identify the first occurrence of errors (timing of failure onset)\n"
            "6. Search for specific patterns like connection errors, OOM, permission issues\n"
            "Report the most critical log findings with exact error messages."
        ),
        expected_output=(
            "Log analysis with: error categories and frequencies, stack traces, "
            "first error timestamps, and the key log lines that reveal root cause."
        ),
        agent=log_agent,
        context=[task_triage, task_metrics],
    )

    task_infra = Task(
        description=(
            "Investigate Kubernetes infrastructure constraints.\n"
            "1. Describe the most problematic pods for full detail\n"
            "2. Check resource quotas — is any namespace at/near quota limits?\n"
            "3. Check PVC status — any unbound volumes?\n"
            "4. Check HPA status — are any HPAs at max replicas?\n"
            "5. Check for node taints or cordon status affecting scheduling\n"
            "6. Look for init container failures\n"
            "Determine if the root cause is infrastructure (quotas, storage, scheduling) "
            "vs application (code bug, config error, resource limits too low)."
        ),
        expected_output=(
            "Infrastructure diagnosis: quota status, PVC health, HPA analysis, "
            "node conditions, and conclusion on whether root cause is infra or application."
        ),
        agent=infra_agent,
        context=[task_triage, task_metrics, task_logs],
    )

    task_runbook = Task(
        description=(
            "Based on the confirmed root cause, create the remediation plan.\n"
            "1. List available runbooks and select the best match for the issue\n"
            "2. Load the selected runbook\n"
            "3. Adapt the runbook steps to the specific pods/deployments affected\n"
            "4. Produce a numbered remediation plan with exact commands\n"
            "5. Mark each step: is_destructive, is_automated\n"
            "6. Order steps from safest to most impactful\n"
            "Always prefer: rolling restart > force delete > scale > cordon\n"
            f"DRY RUN MODE: {'ENABLED' if config.dry_run else 'DISABLED'}"
        ),
        expected_output=(
            "Numbered remediation plan with exact kubectl commands, risk levels, "
            "and which runbook was matched."
        ),
        agent=runbook_agent,
        context=[task_triage, task_metrics, task_logs, task_infra],
    )

    task_command = Task(
        description=(
            "Synthesize ALL findings from the triage, metrics, log, infra, and runbook agents.\n"
            "Produce the final incident report as VALID JSON ONLY (no markdown, no explanation).\n\n"
            "CRITICAL: If no real issues exist, set has_issue=false and severity=SEV4.\n"
            "Only report issues that are CONFIRMED by at least 2 data sources.\n"
            "Do NOT hallucinate metrics, pods, or events not present in the data.\n\n"
            "Output EXACTLY this JSON structure:\n"
            "{\n"
            '  "has_issue": true/false,\n'
            '  "severity": "SEV1|SEV2|SEV3|SEV4",\n'
            '  "title": "concise incident title",\n'
            '  "category": "crashloop|oom|high_cpu|high_memory|image_pull|pending|'
            'node_not_ready|disk_pressure|error_rate|latency|healthy",\n'
            '  "affected_namespaces": ["ns1", "ns2"],\n'
            '  "affected_services": [{"service_name": "x", "namespace": "y", "impact_level": "degraded"}],\n'
            '  "root_cause": "3-5 sentence explanation of what is happening and why",\n'
            '  "contributing_factors": ["factor1", "factor2"],\n'
            '  "confidence_score": 85,\n'
            '  "remediation_plan": [\n'
            '    {"step_number": 1, "description": "...", "command": "kubectl ...", "is_destructive": false, "is_automated": false}\n'
            '  ],\n'
            '  "runbook_used": "crash_loop.yaml"\n'
            "}"
        ),
        expected_output="Valid JSON incident report. No markdown. No explanation. Just the JSON object.",
        agent=commander_agent,
        context=[task_triage, task_metrics, task_logs, task_infra, task_runbook],
    )

    return Crew(
        agents=[triage_agent, metrics_agent, log_agent, infra_agent, runbook_agent, commander_agent],
        tasks=[task_triage, task_metrics, task_logs, task_infra, task_runbook, task_command],
        process=Process.sequential,
        verbose=True,
        memory=False,
        max_rpm=30,  # Throttle to 1 req / 2 sec for free-tier APIs (Mistral)
    )


def run_scan() -> IncidentReport:
    """Execute one full SRE scan and return a structured IncidentReport.
    Retries up to MAX_RETRIES times on rate limit (429) errors.
    """
    start_time = time.time()
    logger.info("Starting AI-SRE scan at %s", datetime.utcnow().isoformat())

    last_error = None
    for attempt in range(MAX_RETRIES):
        try:
            crew = build_crew()
            result = crew.kickoff()
            duration = time.time() - start_time
            raw = result.raw if hasattr(result, "raw") else str(result)
            report = _parse_incident_json(raw, config.llm_model, duration)
            report.detected_at = datetime.utcnow()
            logger.info(
                "Scan complete in %.1fs — %s [%s] confidence=%d%%",
                duration, report.id, report.severity.value, report.confidence_score
            )
            return report
        except Exception as e:
            last_error = e
            err_str = str(e).lower()
            is_rate_limit = any(x in err_str for x in ["rate limit", "429", "ratelimit", "quota"])
            if is_rate_limit and attempt < MAX_RETRIES - 1:
                wait = RETRY_BACKOFF[attempt]
                logger.warning(
                    "Rate limit hit on attempt %d/%d — retrying in %ds",
                    attempt + 1, MAX_RETRIES, wait
                )
                print(f"\n⚠️  Rate limit hit — waiting {wait}s before retry {attempt + 2}/{MAX_RETRIES}...\n")
                time.sleep(wait)
                continue
            raise

    raise last_error

