"""
alerts.py — Alert dispatcher for Claritty AI-SRE.

Supports:
  - Slack (Block Kit format)
  - Generic webhook (POST JSON)
  - Local alert log file

Deduplication: same incident won't re-fire within DEDUP_WINDOW_MINUTES.
"""

import json
import logging
import time
from datetime import datetime, timedelta
from pathlib import Path
from typing import Optional, Dict, Set

import requests

from .incident import IncidentReport, Severity
from .config import config

logger = logging.getLogger("claritty.alerts")

# In-memory dedup tracker: incident_id → last_alerted_timestamp
_alerted_at: Dict[str, float] = {}
DEDUP_WINDOW_MINUTES = 30


def _is_deduped(incident_id: str) -> bool:
    last = _alerted_at.get(incident_id)
    if last is None:
        return False
    return (time.time() - last) < (DEDUP_WINDOW_MINUTES * 60)


def _mark_alerted(incident_id: str) -> None:
    _alerted_at[incident_id] = time.time()


# ─── Severity → Emoji mapping ──────────────────────────────
SEV_EMOJI = {
    Severity.SEV1: "🔴",
    Severity.SEV2: "🟠",
    Severity.SEV3: "🟡",
    Severity.SEV4: "🟢",
}

SEV_COLOR = {
    Severity.SEV1: "#FF0000",
    Severity.SEV2: "#FF8C00",
    Severity.SEV3: "#FFD700",
    Severity.SEV4: "#2ECC40",
}


def _build_slack_blocks(report: IncidentReport) -> dict:
    """Build a rich Slack Block Kit message for an incident."""
    emoji = SEV_EMOJI.get(report.severity, "⚪")
    color = SEV_COLOR.get(report.severity, "#808080")

    # Evidence summary (top 3)
    evidence_lines = []
    for ev in report.evidence[:3]:
        evidence_lines.append(f"• *{ev.type.value.upper()}* ({ev.source}): {ev.description[:100]}")
    evidence_text = "\n".join(evidence_lines) or "_No evidence collected_"

    # Remediation steps (top 3)
    steps = []
    for step in report.remediation_plan[:3]:
        cmd = f"\n  `{step.command}`" if step.command else ""
        steps.append(f"{step.step_number}. {step.description}{cmd}")
    remediation_text = "\n".join(steps) or "_No remediation steps_"

    namespaces = ", ".join(f"`{ns}`" for ns in report.affected_namespaces) or "`default`"
    services = ", ".join(s.service_name for s in report.affected_services[:5]) or "—"

    payload = {
        "attachments": [
            {
                "color": color,
                "blocks": [
                    {
                        "type": "header",
                        "text": {
                            "type": "plain_text",
                            "text": f"{emoji} [{report.severity.value}] {report.title}",
                        },
                    },
                    {
                        "type": "section",
                        "fields": [
                            {"type": "mrkdwn", "text": f"*Incident ID*\n`{report.id}`"},
                            {"type": "mrkdwn", "text": f"*Category*\n{report.category}"},
                            {"type": "mrkdwn", "text": f"*Namespaces*\n{namespaces}"},
                            {"type": "mrkdwn", "text": f"*Services*\n{services}"},
                            {"type": "mrkdwn", "text": f"*Confidence*\n{report.confidence_score}%"},
                            {"type": "mrkdwn", "text": f"*Detected*\n{report.detected_at.strftime('%Y-%m-%d %H:%M UTC')}"},
                        ],
                    },
                    {"type": "divider"},
                    {
                        "type": "section",
                        "text": {
                            "type": "mrkdwn",
                            "text": f"*🔍 Root Cause*\n{report.root_cause[:500]}",
                        },
                    },
                    {
                        "type": "section",
                        "text": {
                            "type": "mrkdwn",
                            "text": f"*📊 Evidence*\n{evidence_text}",
                        },
                    },
                    {
                        "type": "section",
                        "text": {
                            "type": "mrkdwn",
                            "text": f"*🔧 Remediation Plan*\n{remediation_text}",
                        },
                    },
                    {
                        "type": "context",
                        "elements": [
                            {
                                "type": "mrkdwn",
                                "text": f"Claritty AI-SRE | Model: `{report.llm_model}` | "
                                        f"Runbook: `{report.runbook_used or 'none'}`",
                            }
                        ],
                    },
                ],
            }
        ]
    }
    return payload


def send_slack_alert(report: IncidentReport) -> bool:
    """Send incident to Slack via webhook. Returns True on success."""
    if not config.slack_webhook_url:
        logger.debug("Slack webhook not configured, skipping")
        return False

    try:
        payload = _build_slack_blocks(report)
        resp = requests.post(
            config.slack_webhook_url,
            json=payload,
            timeout=10,
        )
        resp.raise_for_status()
        logger.info("Slack alert sent for %s", report.id)
        return True
    except Exception as e:
        logger.error("Slack alert failed for %s: %s", report.id, e)
        return False


def send_webhook_alert(report: IncidentReport) -> bool:
    """Send incident JSON to a generic webhook endpoint."""
    if not config.alert_webhook_url:
        return False
    try:
        resp = requests.post(
            config.alert_webhook_url,
            json=report.model_dump(mode="json"),
            headers={"Content-Type": "application/json", "X-Claritty-Severity": report.severity.value},
            timeout=10,
        )
        resp.raise_for_status()
        logger.info("Webhook alert sent for %s", report.id)
        return True
    except Exception as e:
        logger.error("Webhook alert failed for %s: %s", report.id, e)
        return False


def write_alert_log(report: IncidentReport) -> None:
    """Append incident to a local alert log file."""
    log_path = Path(config.db_path).parent / "alerts.log"
    entry = {
        "timestamp": datetime.utcnow().isoformat(),
        "incident_id": report.id,
        "severity": report.severity.value,
        "title": report.title,
        "category": report.category,
        "namespaces": report.affected_namespaces,
        "root_cause": report.root_cause[:200],
    }
    with open(log_path, "a") as f:
        f.write(json.dumps(entry) + "\n")


def dispatch_alerts(report: IncidentReport) -> None:
    """
    Main entry point: dispatch all configured alerts for an incident.
    Respects severity filter and deduplication.
    """
    if report.severity.value not in config.alert_on_severities:
        logger.debug(
            "Severity %s not in alert list %s, skipping",
            report.severity.value,
            config.alert_on_severities,
        )
        return

    if _is_deduped(report.id):
        logger.debug("Incident %s already alerted within dedup window", report.id)
        return

    logger.info("Dispatching alerts for %s [%s]", report.id, report.severity.value)

    send_slack_alert(report)
    send_webhook_alert(report)
    write_alert_log(report)

    _mark_alerted(report.id)
