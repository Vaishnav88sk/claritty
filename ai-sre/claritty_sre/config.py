"""
config.py — Centralized configuration for Claritty AI-SRE Engine.

All settings are loaded from environment variables (via .env),
with sensible production defaults.
"""

import os
from dataclasses import dataclass, field
from typing import List, Optional
from dotenv import load_dotenv

load_dotenv()


@dataclass
class SREConfig:
    # ─── LLM ───────────────────────────────────────────────
    llm_provider: str = os.getenv("LLM_PROVIDER", "mistral")
    llm_model: str = os.getenv("LLM_MODEL", "mistral/mistral-large-latest")
    llm_temperature: float = float(os.getenv("LLM_TEMPERATURE", "0.1"))
    llm_max_tokens: int = int(os.getenv("LLM_MAX_TOKENS", "2048"))

    # ─── Kubernetes ────────────────────────────────────────
    namespaces: List[str] = field(
        default_factory=lambda: os.getenv("K8S_NAMESPACES", "default").split(",")
    )
    scan_all_namespaces: bool = os.getenv("K8S_SCAN_ALL", "false").lower() == "true"
    kubeconfig_path: Optional[str] = os.getenv("KUBECONFIG", None)

    # ─── Prometheus ────────────────────────────────────────
    prometheus_url: str = os.getenv("PROMETHEUS_URL", "http://localhost:9090")
    prometheus_enabled: bool = os.getenv("PROMETHEUS_ENABLED", "true").lower() == "true"
    metrics_window: str = os.getenv("METRICS_WINDOW", "5m")

    # ─── Loki ──────────────────────────────────────────────
    loki_url: str = os.getenv("LOKI_URL", "http://localhost:3100")
    loki_enabled: bool = os.getenv("LOKI_ENABLED", "false").lower() == "true"

    # ─── Alerting ──────────────────────────────────────────
    slack_webhook_url: Optional[str] = os.getenv("SLACK_WEBHOOK_URL", None)
    alert_webhook_url: Optional[str] = os.getenv("ALERT_WEBHOOK_URL", None)
    alert_on_severities: List[str] = field(
        default_factory=lambda: os.getenv("ALERT_SEVERITIES", "SEV1,SEV2").split(",")
    )

    # ─── Severity Thresholds ───────────────────────────────
    cpu_warning_pct: float = float(os.getenv("CPU_WARNING_PCT", "80.0"))
    cpu_critical_pct: float = float(os.getenv("CPU_CRITICAL_PCT", "95.0"))
    memory_warning_pct: float = float(os.getenv("MEMORY_WARNING_PCT", "85.0"))
    memory_critical_pct: float = float(os.getenv("MEMORY_CRITICAL_PCT", "95.0"))
    restart_warning_count: int = int(os.getenv("RESTART_WARNING_COUNT", "5"))
    restart_critical_count: int = int(os.getenv("RESTART_CRITICAL_COUNT", "15"))
    error_rate_warning_pct: float = float(os.getenv("ERROR_RATE_WARNING_PCT", "1.0"))
    error_rate_critical_pct: float = float(os.getenv("ERROR_RATE_CRITICAL_PCT", "5.0"))

    # ─── Scan Settings ─────────────────────────────────────
    scan_interval_seconds: int = int(os.getenv("SCAN_INTERVAL_SECONDS", "300"))
    pod_log_tail_lines: int = int(os.getenv("POD_LOG_TAIL_LINES", "100"))
    max_events_per_scan: int = int(os.getenv("MAX_EVENTS_PER_SCAN", "50"))
    agent_timeout_seconds: int = int(os.getenv("AGENT_TIMEOUT_SECONDS", "120"))

    # ─── Runbooks ──────────────────────────────────────────
    runbooks_dir: str = os.getenv(
        "RUNBOOKS_DIR",
        os.path.join(os.path.dirname(os.path.dirname(__file__)), "runbooks")
    )
    dry_run: bool = os.getenv("DRY_RUN", "true").lower() == "true"
    auto_remediate: bool = os.getenv("AUTO_REMEDIATE", "false").lower() == "true"

    # ─── Database ──────────────────────────────────────────
    db_path: str = os.getenv(
        "DB_PATH",
        os.path.join(os.path.dirname(os.path.dirname(__file__)), "claritty_sre.db")
    )

    # ─── API Keys ──────────────────────────────────────────
    groq_api_key: Optional[str] = os.getenv("GROQ_API_KEY")
    mistral_api_key: Optional[str] = os.getenv("MISTRAL_API_KEY")
    openai_api_key: Optional[str] = os.getenv("OPENAI_API_KEY")

    def validate(self) -> None:
        """Validate critical config values and set env vars for LLM providers."""
        if self.llm_provider == "groq" and not self.groq_api_key:
            raise ValueError("GROQ_API_KEY is required when LLM_PROVIDER=groq")
        if self.llm_provider == "mistral" and not self.mistral_api_key:
            raise ValueError("MISTRAL_API_KEY is required when LLM_PROVIDER=mistral")
        if self.llm_provider == "openai" and not self.openai_api_key:
            raise ValueError("OPENAI_API_KEY is required when LLM_PROVIDER=openai")

        # Export for LiteLLM / CrewAI
        if self.groq_api_key:
            os.environ["GROQ_API_KEY"] = self.groq_api_key
        if self.mistral_api_key:
            os.environ["MISTRAL_API_KEY"] = self.mistral_api_key
        if self.openai_api_key:
            os.environ["OPENAI_API_KEY"] = self.openai_api_key

        # Silence CrewAI telemetry
        os.environ["CREWAI_TRACING_ENABLED"] = "false"
        os.environ["CREWAI_DISABLE_TRACING"] = "true"
        os.environ.setdefault("OTEL_SDK_DISABLED", "true")


# ─── Singleton ─────────────────────────────────────────────
config = SREConfig()
