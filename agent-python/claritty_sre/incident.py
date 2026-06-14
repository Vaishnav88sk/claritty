"""
incident.py — Structured incident data models for Claritty AI-SRE.

Uses Pydantic for strict validation. The IncidentReport is the core
output type produced by the agent pipeline.
"""

from __future__ import annotations
import uuid
from datetime import datetime
from enum import Enum
from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field


class Severity(str, Enum):
    SEV1 = "SEV1"   # Critical — service down, data loss risk
    SEV2 = "SEV2"   # High — major degradation, significant impact
    SEV3 = "SEV3"   # Medium — partial degradation, workaround exists
    SEV4 = "SEV4"   # Low — minor issue, no user impact


class IncidentStatus(str, Enum):
    OPEN = "OPEN"
    INVESTIGATING = "INVESTIGATING"
    MITIGATED = "MITIGATED"
    RESOLVED = "RESOLVED"
    IGNORED = "IGNORED"


class EvidenceType(str, Enum):
    METRIC = "metric"
    LOG = "log"
    EVENT = "event"
    NODE = "node"
    RUNBOOK = "runbook"


class Evidence(BaseModel):
    """A single piece of evidence supporting the incident analysis."""
    type: EvidenceType
    source: str                          # e.g. "prometheus", "kubectl logs", "k8s events"
    description: str                     # Human-readable summary
    raw: Optional[str] = None            # Raw data (truncated)
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    metric_name: Optional[str] = None
    metric_value: Optional[float] = None
    pod_name: Optional[str] = None
    namespace: Optional[str] = None


class RemediationStep(BaseModel):
    """A single step in a remediation plan."""
    step_number: int
    description: str
    command: Optional[str] = None        # Exact kubectl/shell command
    is_destructive: bool = False         # Requires extra confirmation
    is_automated: bool = False           # Can auto-execute?
    status: str = "PENDING"             # PENDING / APPLIED / SKIPPED / FAILED
    applied_at: Optional[datetime] = None
    result: Optional[str] = None


class ServiceImpact(BaseModel):
    """Describes the impact on a specific service/workload."""
    service_name: str
    namespace: str
    impact_level: str                   # "down" / "degraded" / "at_risk"
    affected_pods: List[str] = []
    error_rate: Optional[float] = None
    latency_p99_ms: Optional[float] = None


class IncidentReport(BaseModel):
    """
    Full structured incident report produced by the AI-SRE pipeline.
    This is persisted to SQLite and can be exported as JSON.
    """
    # ─── Identity ───────────────────────────────────────────
    id: str = Field(default_factory=lambda: f"INC-{uuid.uuid4().hex[:8].upper()}")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

    # ─── Classification ─────────────────────────────────────
    severity: Severity
    title: str
    category: str = "unknown"           # "oom", "crashloop", "high_cpu", etc.
    status: IncidentStatus = IncidentStatus.OPEN

    # ─── Scope ──────────────────────────────────────────────
    affected_namespaces: List[str] = []
    affected_services: List[ServiceImpact] = []
    affected_pod_count: int = 0
    cluster_health_score: Optional[float] = None   # 0.0–100.0

    # ─── Analysis ───────────────────────────────────────────
    root_cause: str = ""
    contributing_factors: List[str] = []
    evidence: List[Evidence] = []
    confidence_score: int = 0           # 0–100

    # ─── Remediation ────────────────────────────────────────
    remediation_plan: List[RemediationStep] = []
    runbook_used: Optional[str] = None
    auto_remediated: bool = False

    # ─── Timing ─────────────────────────────────────────────
    detected_at: datetime = Field(default_factory=datetime.utcnow)
    mitigated_at: Optional[datetime] = None
    resolved_at: Optional[datetime] = None
    mttr_seconds: Optional[int] = None

    # ─── Metadata ───────────────────────────────────────────
    llm_model: str = ""
    scan_duration_seconds: Optional[float] = None
    raw_agent_output: Optional[str] = None

    def compute_mttr(self) -> None:
        """Compute MTTR if incident is resolved/mitigated."""
        end_time = self.resolved_at or self.mitigated_at
        if end_time:
            self.mttr_seconds = int((end_time - self.detected_at).total_seconds())

    def severity_color(self) -> str:
        """Return Rich color for this severity."""
        colors = {
            Severity.SEV1: "bold red",
            Severity.SEV2: "bold orange1",
            Severity.SEV3: "bold yellow",
            Severity.SEV4: "bold green",
        }
        return colors.get(self.severity, "white")

    def to_summary_dict(self) -> Dict[str, Any]:
        """Compact dict for table/list display."""
        return {
            "id": self.id,
            "severity": self.severity.value,
            "title": self.title[:60] + ("…" if len(self.title) > 60 else ""),
            "status": self.status.value,
            "namespaces": ", ".join(self.affected_namespaces) or "—",
            "confidence": f"{self.confidence_score}%",
            "created_at": self.created_at.strftime("%Y-%m-%d %H:%M:%S"),
        }


class ClusterHealthSnapshot(BaseModel):
    """Point-in-time cluster health for trend tracking."""
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    total_nodes: int = 0
    ready_nodes: int = 0
    total_pods: int = 0
    running_pods: int = 0
    pending_pods: int = 0
    failed_pods: int = 0
    crashloop_pods: int = 0
    evicted_pods: int = 0
    cpu_usage_pct: float = 0.0
    memory_usage_pct: float = 0.0
    open_incidents: int = 0
    health_score: float = 100.0

    def compute_health_score(self) -> None:
        """Heuristic health score 0–100."""
        score = 100.0
        if self.total_nodes > 0:
            node_health = (self.ready_nodes / self.total_nodes) * 100
            score -= (100 - node_health) * 0.4
        if self.total_pods > 0:
            pod_health = (self.running_pods / self.total_pods) * 100
            score -= (100 - pod_health) * 0.3
        score -= min(self.cpu_usage_pct * 0.1, 15)
        score -= min(self.memory_usage_pct * 0.1, 15)
        score -= self.open_incidents * 5
        self.health_score = max(0.0, min(100.0, score))
