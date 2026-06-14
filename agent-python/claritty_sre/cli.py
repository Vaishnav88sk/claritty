#!/usr/bin/env python3
"""
cli.py — Claritty AI-SRE Engine CLI
======================================
Production-grade, multi-agent Site Reliability Engineering platform
for Kubernetes clusters.

Usage:
  clarctl scan              Run a single SRE scan
  clarctl watch             Continuous monitoring loop with dashboard
  clarctl incidents         View incident history
  clarctl show <id>         Show detailed incident report
  clarctl apply <id>        Apply remediation for an incident
  clarctl report <id>       Export incident as JSON report
  clarctl status            Show cluster health snapshot
"""

import json
import logging
import signal
import sys
import time
from datetime import datetime
from pathlib import Path

import click
from rich.console import Console
from rich.panel import Panel
from rich.text import Text

from claritty_sre.config import config
from claritty_sre import db, alerts
from claritty_sre.dashboard import (
    console, print_banner, render_cluster_health,
    render_incidents_table, render_incident_detail,
    render_mttr_stats, make_scan_progress, prompt_apply_fix,
)
from claritty_sre.incident import (
    IncidentReport, IncidentStatus, ClusterHealthSnapshot,
    Severity,
)
from claritty_sre.pipeline import run_scan

# ─── Logging Setup ────────────────────────────────────────
logging.basicConfig(
    level=logging.WARNING,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[
        logging.FileHandler(Path(config.db_path).parent / "claritty_sre.log"),
    ],
)
logger = logging.getLogger("claritty")


def _build_snapshot_from_report(report: IncidentReport) -> ClusterHealthSnapshot:
    """Build a cluster health snapshot from an incident report for persistence."""
    snapshot = ClusterHealthSnapshot(
        timestamp=datetime.utcnow(),
        open_incidents=db.get_open_incident_count(),
    )
    snapshot.compute_health_score()
    return snapshot


def _do_scan(dry_run_override: bool = False) -> IncidentReport:
    """Run one scan cycle with progress display."""
    progress = make_scan_progress()
    with progress:
        task = progress.add_task("Initializing agents…", total=None)

        progress.update(task, description="[cyan]🔍 Triage Agent scanning cluster…[/cyan]")
        report = run_scan()

    return report


def _handle_report(report: IncidentReport, apply: bool = False) -> None:
    """Display and persist an incident report."""
    # Persist
    db.save_incident(report)

    # Save snapshot
    snapshot = _build_snapshot_from_report(report)
    db.save_snapshot(snapshot)

    # Display
    render_incident_detail(report)

    # Alert dispatch
    if report.has_issue if hasattr(report, 'has_issue') else report.severity != Severity.SEV4:
        alerts.dispatch_alerts(report)

    # Prompt for remediation
    if apply and report.remediation_plan:
        action = prompt_apply_fix(report)
        if action == "execute":
            config.dry_run = False
            _apply_remediation(report, dry_run=False)
        elif action == "dry":
            _apply_remediation(report, dry_run=True)


def _apply_remediation(report: IncidentReport, dry_run: bool = True) -> None:
    """Execute remediation steps for an incident."""
    from claritty_sre.tools.runbook_tools import _run_command
    console.print()
    for step in report.remediation_plan:
        if step.command:
            console.print(f"  [bold]Step {step.step_number}:[/bold] {step.description}")
            result = _run_command(step.command, dry_run=dry_run)
            if result.get("dry_run"):
                console.print(f"    [dim][DRY RUN] {step.command}[/dim]")
            elif result.get("success"):
                console.print(f"    [green]✓ Done[/green]")
                step.status = "APPLIED"
                step.applied_at = datetime.utcnow()
                step.result = result.get("stdout", "")
            else:
                console.print(f"    [red]✗ Failed: {result.get('error') or result.get('stderr')}[/red]")
                step.status = "FAILED"

    # Update incident status
    applied = [s for s in report.remediation_plan if s.status == "APPLIED"]
    if applied:
        report.status = IncidentStatus.MITIGATED
        report.mitigated_at = datetime.utcnow()
        report.compute_mttr()
        db.save_incident(report)
        db.update_incident_status(report.id, IncidentStatus.MITIGATED)
        console.print(f"\n[green bold]✓ Incident {report.id} marked as MITIGATED[/green bold]")


# ─── CLI Commands ─────────────────────────────────────────

@click.group()
@click.option("--debug", is_flag=True, help="Enable debug logging")
def cli(debug: bool):
    """Claritty AI-SRE — Production-grade Kubernetes observability engine."""
    if debug:
        logging.getLogger("claritty").setLevel(logging.DEBUG)
        logging.getLogger().addHandler(logging.StreamHandler())

    # Init DB
    db.init_db()
    config.validate()


@cli.command()
@click.option("--apply", is_flag=True, help="Prompt to apply remediation after scan")
@click.option("--dry-run/--no-dry-run", default=True, help="Dry run mode (default: enabled)")
def scan(apply: bool, dry_run: bool):
    """Run a single AI-SRE scan across the cluster."""
    print_banner()

    console.print(Panel(
        Text.assemble(
            ("Namespaces: ", "bright_black"),
            (", ".join(config.namespaces), "cyan"),
            ("   LLM: ", "bright_black"),
            (config.llm_model, "cyan"),
            ("   Dry Run: ", "bright_black"),
            (str(dry_run), "yellow" if dry_run else "red"),
        ),
        border_style="bright_black",
    ))

    console.print()
    console.print("[cyan bold]Starting AI-SRE scan...[/cyan bold]")
    console.print("[bright_black]This may take 1-3 minutes depending on cluster size.[/bright_black]")
    console.print()

    try:
        report = _do_scan(dry_run_override=dry_run)
        _handle_report(report, apply=apply)
    except KeyboardInterrupt:
        console.print("\n[yellow]Scan interrupted by user.[/yellow]")
    except Exception as e:
        console.print(f"\n[red bold]Scan failed: {e}[/red bold]")
        logger.exception("Scan failed")
        raise SystemExit(1)


@cli.command()
@click.option("--interval", default=config.scan_interval_seconds, show_default=True,
              help="Scan interval in seconds")
@click.option("--apply", is_flag=True, help="Auto-prompt remediation after each scan")
def watch(interval: int, apply: bool):
    """Continuous monitoring loop with live dashboard."""
    print_banner()

    console.print(f"[cyan bold]Starting continuous SRE watcher[/cyan bold] "
                  f"[bright_black](interval: {interval}s, Ctrl+C to stop)[/bright_black]")
    console.print()

    scan_count = 0
    last_snapshot: ClusterHealthSnapshot | None = None
    last_incidents: list[IncidentReport] = []

    def graceful_exit(sig, frame):
        console.print("\n[yellow]Watcher stopped by user.[/yellow]")
        sys.exit(0)

    signal.signal(signal.SIGINT, graceful_exit)
    signal.signal(signal.SIGTERM, graceful_exit)

    while True:
        scan_count += 1
        ts = datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC")
        console.rule(f"[cyan]Scan #{scan_count}  ·  {ts}[/cyan]")

        try:
            report = _do_scan()
            db.save_incident(report)

            snapshot = _build_snapshot_from_report(report)
            db.save_snapshot(snapshot)
            last_snapshot = snapshot

            # Refresh incidents
            last_incidents = db.get_incidents(limit=10, hours=24)

            # Render dashboard panels
            console.print(render_cluster_health(last_snapshot))
            console.print(render_incidents_table(last_incidents))
            console.print(render_mttr_stats(db.get_mttr_stats()))

            # Show detail for new issues
            if report.severity in (Severity.SEV1, Severity.SEV2):
                render_incident_detail(report)
                alerts.dispatch_alerts(report)
                if apply:
                    action = prompt_apply_fix(report)
                    if action == "execute":
                        config.dry_run = False
                        _apply_remediation(report, dry_run=False)
                    elif action == "dry":
                        _apply_remediation(report, dry_run=True)
            else:
                console.print(
                    Panel(
                        f"[green bold]✓ {report.title}[/green bold]\n"
                        f"[dim]Confidence: {report.confidence_score}% · ID: {report.id}[/dim]",
                        border_style="green",
                    )
                )
        except Exception as e:
            console.print(f"[red]Scan #{scan_count} failed: {e}[/red]")
            logger.exception("Watch scan failed")

        console.print(f"\n[dim]Next scan in {interval}s...[/dim]\n")
        time.sleep(interval)


@cli.command()
@click.option("--severity", default="", help="Filter by severity (SEV1, SEV2, SEV3, SEV4)")
@click.option("--status", default="", help="Filter by status (OPEN, INVESTIGATING, MITIGATED, RESOLVED)")
@click.option("--hours", default=24, show_default=True, help="Look back N hours")
@click.option("--limit", default=20, show_default=True, help="Max results to show")
def incidents(severity: str, status: str, hours: int, limit: int):
    """View incident history with optional filters."""
    print_banner()

    inc_list = db.get_incidents(
        severity=severity or None,
        status=status or None,
        limit=limit,
        hours=hours,
    )
    console.print(render_incidents_table(inc_list))
    console.print(render_mttr_stats(db.get_mttr_stats()))


@cli.command()
@click.argument("incident_id")
def show(incident_id: str):
    """Show detailed view of a specific incident."""
    report = db.get_incident_by_id(incident_id)
    if not report:
        console.print(f"[red]Incident '{incident_id}' not found.[/red]")
        raise SystemExit(1)
    print_banner()
    render_incident_detail(report)


@cli.command()
@click.argument("incident_id")
@click.option("--dry-run/--no-dry-run", default=True, help="Dry run mode (default: enabled)")
def apply(incident_id: str, dry_run: bool):
    """Apply the remediation plan for a specific incident."""
    report = db.get_incident_by_id(incident_id)
    if not report:
        console.print(f"[red]Incident '{incident_id}' not found.[/red]")
        raise SystemExit(1)

    render_incident_detail(report)

    if not report.remediation_plan:
        console.print("[yellow]No remediation steps available for this incident.[/yellow]")
        raise SystemExit(0)

    action = prompt_apply_fix(report)
    if action == "execute":
        config.dry_run = False
        _apply_remediation(report, dry_run=False)
    elif action == "dry":
        _apply_remediation(report, dry_run=True)


@cli.command()
@click.argument("incident_id")
@click.option("--output", "-o", default="", help="Output file path (default: stdout)")
def report(incident_id: str, output: str):
    """Export a full incident report as JSON."""
    inc = db.get_incident_by_id(incident_id)
    if not inc:
        console.print(f"[red]Incident '{incident_id}' not found.[/red]")
        raise SystemExit(1)

    json_str = inc.model_dump_json(indent=2)
    if output:
        Path(output).write_text(json_str)
        console.print(f"[green]Report saved to {output}[/green]")
    else:
        console.print_json(json_str)


@cli.command()
def status():
    """Show current cluster health snapshot from DB."""
    print_banner()
    snapshots = db.get_recent_snapshots(limit=1)
    snapshot = snapshots[0] if snapshots else None
    console.print(render_cluster_health(snapshot))

    open_inc = db.get_incidents(status="OPEN", limit=5)
    if open_inc:
        console.print("\n[red bold]⚠  Open Incidents:[/red bold]")
        console.print(render_incidents_table(open_inc))
    else:
        console.print("\n[green bold]✓ No open incidents.[/green bold]")


if __name__ == "__main__":
    cli()
