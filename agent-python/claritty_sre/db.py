"""
db.py — SQLite persistence layer for Claritty AI-SRE.

Stores incident reports and cluster health snapshots for:
  - Historical trend analysis
  - MTTR tracking
  - Incident deduplication
  - Report export
"""

import sqlite3
import json
import logging
from datetime import datetime, timedelta
from typing import List, Optional
from contextlib import contextmanager

from .incident import IncidentReport, ClusterHealthSnapshot, Severity, IncidentStatus
from .config import config

logger = logging.getLogger("claritty.db")


@contextmanager
def get_conn():
    """Context manager for SQLite connections."""
    conn = sqlite3.connect(config.db_path)
    conn.row_factory = sqlite3.Row
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def init_db() -> None:
    """Create tables if they don't exist."""
    with get_conn() as conn:
        conn.executescript("""
            CREATE TABLE IF NOT EXISTS incidents (
                id              TEXT PRIMARY KEY,
                created_at      TEXT NOT NULL,
                updated_at      TEXT NOT NULL,
                severity        TEXT NOT NULL,
                title           TEXT NOT NULL,
                category        TEXT DEFAULT 'unknown',
                status          TEXT NOT NULL DEFAULT 'OPEN',
                affected_namespaces TEXT DEFAULT '[]',
                affected_pod_count  INTEGER DEFAULT 0,
                root_cause      TEXT DEFAULT '',
                contributing_factors TEXT DEFAULT '[]',
                confidence_score    INTEGER DEFAULT 0,
                runbook_used    TEXT,
                auto_remediated INTEGER DEFAULT 0,
                detected_at     TEXT,
                mitigated_at    TEXT,
                resolved_at     TEXT,
                mttr_seconds    INTEGER,
                llm_model       TEXT,
                scan_duration_seconds REAL,
                raw_json        TEXT NOT NULL
            );

            CREATE TABLE IF NOT EXISTS cluster_snapshots (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp       TEXT NOT NULL,
                total_nodes     INTEGER,
                ready_nodes     INTEGER,
                total_pods      INTEGER,
                running_pods    INTEGER,
                pending_pods    INTEGER,
                failed_pods     INTEGER,
                crashloop_pods  INTEGER,
                cpu_usage_pct   REAL,
                memory_usage_pct REAL,
                open_incidents  INTEGER,
                health_score    REAL
            );

            CREATE INDEX IF NOT EXISTS idx_incidents_severity
                ON incidents(severity);
            CREATE INDEX IF NOT EXISTS idx_incidents_status
                ON incidents(status);
            CREATE INDEX IF NOT EXISTS idx_incidents_created
                ON incidents(created_at);
            CREATE INDEX IF NOT EXISTS idx_snapshots_timestamp
                ON cluster_snapshots(timestamp);
        """)
    logger.info("Database initialized at %s", config.db_path)


def save_incident(report: IncidentReport) -> None:
    """Persist an incident report to the database."""
    with get_conn() as conn:
        conn.execute("""
            INSERT OR REPLACE INTO incidents
            (id, created_at, updated_at, severity, title, category, status,
             affected_namespaces, affected_pod_count, root_cause,
             contributing_factors, confidence_score, runbook_used,
             auto_remediated, detected_at, mitigated_at, resolved_at,
             mttr_seconds, llm_model, scan_duration_seconds, raw_json)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        """, (
            report.id,
            report.created_at.isoformat(),
            datetime.utcnow().isoformat(),
            report.severity.value,
            report.title,
            report.category,
            report.status.value,
            json.dumps(report.affected_namespaces),
            report.affected_pod_count,
            report.root_cause,
            json.dumps(report.contributing_factors),
            report.confidence_score,
            report.runbook_used,
            int(report.auto_remediated),
            report.detected_at.isoformat(),
            report.mitigated_at.isoformat() if report.mitigated_at else None,
            report.resolved_at.isoformat() if report.resolved_at else None,
            report.mttr_seconds,
            report.llm_model,
            report.scan_duration_seconds,
            report.model_dump_json(),
        ))
    logger.debug("Saved incident %s", report.id)


def save_snapshot(snapshot: ClusterHealthSnapshot) -> None:
    """Save a cluster health snapshot."""
    with get_conn() as conn:
        conn.execute("""
            INSERT INTO cluster_snapshots
            (timestamp, total_nodes, ready_nodes, total_pods, running_pods,
             pending_pods, failed_pods, crashloop_pods, cpu_usage_pct,
             memory_usage_pct, open_incidents, health_score)
            VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
        """, (
            snapshot.timestamp.isoformat(),
            snapshot.total_nodes,
            snapshot.ready_nodes,
            snapshot.total_pods,
            snapshot.running_pods,
            snapshot.pending_pods,
            snapshot.failed_pods,
            snapshot.crashloop_pods,
            snapshot.cpu_usage_pct,
            snapshot.memory_usage_pct,
            snapshot.open_incidents,
            snapshot.health_score,
        ))


def get_incidents(
    severity: Optional[str] = None,
    status: Optional[str] = None,
    limit: int = 50,
    hours: int = 0,
) -> List[IncidentReport]:
    """Query incidents with optional filters."""
    clauses = []
    params = []

    if severity:
        clauses.append("severity = ?")
        params.append(severity.upper())
    if status:
        clauses.append("status = ?")
        params.append(status.upper())
    if hours > 0:
        since = (datetime.utcnow() - timedelta(hours=hours)).isoformat()
        clauses.append("created_at >= ?")
        params.append(since)

    where = ("WHERE " + " AND ".join(clauses)) if clauses else ""
    params.append(limit)

    with get_conn() as conn:
        rows = conn.execute(
            f"SELECT raw_json FROM incidents {where} ORDER BY created_at DESC LIMIT ?",
            params,
        ).fetchall()

    reports = []
    for row in rows:
        try:
            reports.append(IncidentReport.model_validate_json(row["raw_json"]))
        except Exception as e:
            logger.warning("Failed to deserialize incident: %s", e)
    return reports


def get_incident_by_id(incident_id: str) -> Optional[IncidentReport]:
    """Get a single incident by ID."""
    with get_conn() as conn:
        row = conn.execute(
            "SELECT raw_json FROM incidents WHERE id = ?", (incident_id,)
        ).fetchone()
    if not row:
        return None
    return IncidentReport.model_validate_json(row["raw_json"])


def update_incident_status(incident_id: str, status: IncidentStatus) -> None:
    """Update incident status (e.g., MITIGATED/RESOLVED)."""
    now = datetime.utcnow().isoformat()
    extra = {}
    if status == IncidentStatus.MITIGATED:
        extra["mitigated_at"] = now
    elif status == IncidentStatus.RESOLVED:
        extra["resolved_at"] = now

    with get_conn() as conn:
        conn.execute(
            "UPDATE incidents SET status = ?, updated_at = ? WHERE id = ?",
            (status.value, now, incident_id),
        )
        if extra:
            for col, val in extra.items():
                conn.execute(
                    f"UPDATE incidents SET {col} = ? WHERE id = ?",
                    (val, incident_id),
                )


def get_open_incident_count() -> int:
    """Count open incidents for deduplication and health scoring."""
    with get_conn() as conn:
        row = conn.execute(
            "SELECT COUNT(*) as c FROM incidents WHERE status IN ('OPEN', 'INVESTIGATING')"
        ).fetchone()
    return row["c"] if row else 0


def get_recent_snapshots(limit: int = 60) -> List[ClusterHealthSnapshot]:
    """Return recent health snapshots for trend display."""
    with get_conn() as conn:
        rows = conn.execute(
            "SELECT * FROM cluster_snapshots ORDER BY timestamp DESC LIMIT ?", (limit,)
        ).fetchall()
    snapshots = []
    for row in rows:
        try:
            snapshots.append(ClusterHealthSnapshot(**dict(row)))
        except Exception:
            pass
    return snapshots


def get_mttr_stats() -> dict:
    """Compute MTTR statistics across all resolved incidents."""
    with get_conn() as conn:
        rows = conn.execute("""
            SELECT severity, AVG(mttr_seconds) as avg_mttr, COUNT(*) as count
            FROM incidents
            WHERE mttr_seconds IS NOT NULL
            GROUP BY severity
        """).fetchall()
    return {
        row["severity"]: {
            "avg_mttr_seconds": int(row["avg_mttr"] or 0),
            "count": row["count"],
        }
        for row in rows
    }


def incident_already_open(category: str, namespace: str) -> bool:
    """
    Check if an open/investigating incident for this category+namespace
    already exists (deduplication guard).
    """
    with get_conn() as conn:
        row = conn.execute("""
            SELECT id FROM incidents
            WHERE category = ?
              AND affected_namespaces LIKE ?
              AND status IN ('OPEN', 'INVESTIGATING')
            LIMIT 1
        """, (category, f'%"{namespace}"%')).fetchone()
    return row is not None
