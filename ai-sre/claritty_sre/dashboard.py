"""
dashboard.py — Rich-powered CLI dashboard for Claritty AI-SRE.

Renders a beautiful, live terminal UI with:
  - Cluster health overview panel
  - Active incidents table (color-coded by severity)
  - Recent scan timeline
  - MTTR statistics
  - Live scan progress
"""

import time
from datetime import datetime
from typing import List, Optional

from rich import box
from rich.align import Align
from rich.columns import Columns
from rich.console import Console
from rich.layout import Layout
from rich.live import Live
from rich.panel import Panel
from rich.progress import Progress, SpinnerColumn, TextColumn, TimeElapsedColumn
from rich.rule import Rule
from rich.style import Style
from rich.table import Table
from rich.text import Text

from .incident import IncidentReport, Severity, IncidentStatus, ClusterHealthSnapshot

console = Console()

# ─── Color Palette ────────────────────────────────────────
SEV_STYLE = {
    Severity.SEV1: "bold white on red",
    Severity.SEV2: "bold black on orange1",
    Severity.SEV3: "bold black on yellow",
    Severity.SEV4: "bold black on green",
}
SEV_COLOR = {
    Severity.SEV1: "red",
    Severity.SEV2: "orange1",
    Severity.SEV3: "yellow",
    Severity.SEV4: "green",
}
STATUS_STYLE = {
    IncidentStatus.OPEN: "bold red",
    IncidentStatus.INVESTIGATING: "bold yellow",
    IncidentStatus.MITIGATED: "bold cyan",
    IncidentStatus.RESOLVED: "bold green",
    IncidentStatus.IGNORED: "dim",
}


def print_banner() -> None:
    """Print the Claritty AI-SRE ASCII banner."""
    console.print()
    console.print(
        Panel.fit(
            Align.center(
                Text.assemble(
                    ("  ██████╗██╗      █████╗ ██████╗ ██╗████████╗████████╗██╗   ██╗\n", "cyan bold"),
                    (" ██╔════╝██║     ██╔══██╗██╔══██╗██║╚══██╔══╝╚══██╔══╝╚██╗ ██╔╝\n", "cyan bold"),
                    (" ██║     ██║     ███████║██████╔╝██║   ██║      ██║    ╚████╔╝ \n", "cyan bold"),
                    (" ██║     ██║     ██╔══██║██╔══██╗██║   ██║      ██║     ╚██╔╝  \n", "cyan bold"),
                    (" ╚██████╗███████╗██║  ██║██║  ██║██║   ██║      ██║      ██║   \n", "cyan bold"),
                    ("  ╚═════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝   ╚═╝      ╚═╝      ╚═╝   \n", "cyan bold"),
                    ("              AI-SRE Engine  ·  v2.0  ·  Kubernetes Observability\n", "bright_black"),
                )
            ),
            border_style="cyan",
            padding=(0, 2),
        )
    )
    console.print()


def render_cluster_health(snapshot: Optional[ClusterHealthSnapshot]) -> Panel:
    """Render the cluster health overview panel."""
    if not snapshot:
        return Panel("[dim]No health data yet[/dim]", title="Cluster Health", border_style="cyan")

    score = snapshot.health_score
    if score >= 90:
        score_color, score_icon = "green", "✓"
    elif score >= 70:
        score_color, score_icon = "yellow", "⚠"
    elif score >= 50:
        score_color, score_icon = "orange1", "!"
    else:
        score_color, score_icon = "red", "✗"

    # Build metrics grid
    grid = Table.grid(padding=(0, 2))
    grid.add_column(style="bright_black", min_width=14)
    grid.add_column(style="white bold")
    grid.add_column(style="bright_black", min_width=14)
    grid.add_column(style="white bold")

    def node_style(ready, total):
        return "green" if ready == total else ("orange1" if ready > total // 2 else "red")

    grid.add_row(
        "Nodes",
        Text(f"{snapshot.ready_nodes}/{snapshot.total_nodes}",
             style=node_style(snapshot.ready_nodes, snapshot.total_nodes)),
        "Running Pods",
        Text(f"{snapshot.running_pods}/{snapshot.total_pods}", style="green" if snapshot.running_pods == snapshot.total_pods else "yellow"),
    )
    grid.add_row(
        "CPU Usage",
        Text(f"{snapshot.cpu_usage_pct:.1f}%",
             style="red" if snapshot.cpu_usage_pct > 90 else ("yellow" if snapshot.cpu_usage_pct > 75 else "green")),
        "Memory Usage",
        Text(f"{snapshot.memory_usage_pct:.1f}%",
             style="red" if snapshot.memory_usage_pct > 90 else ("yellow" if snapshot.memory_usage_pct > 80 else "green")),
    )
    grid.add_row(
        "Pending Pods",
        Text(str(snapshot.pending_pods), style="yellow" if snapshot.pending_pods > 0 else "green"),
        "CrashLoop Pods",
        Text(str(snapshot.crashloop_pods), style="red" if snapshot.crashloop_pods > 0 else "green"),
    )
    grid.add_row(
        "Failed Pods",
        Text(str(snapshot.failed_pods), style="red" if snapshot.failed_pods > 0 else "green"),
        "Open Incidents",
        Text(str(snapshot.open_incidents), style="red" if snapshot.open_incidents > 0 else "green"),
    )

    header = Text.assemble(
        (f" {score_icon} Health Score: ", "white"),
        (f"{score:.0f}/100 ", f"bold {score_color}"),
        (f"  ·  {snapshot.timestamp.strftime('%H:%M:%S UTC')}", "bright_black"),
    )

    return Panel(
        Columns([header, Rule(style="bright_black"), grid]),
        title="[cyan bold]Cluster Health Overview[/cyan bold]",
        border_style="cyan",
        padding=(0, 1),
    )


def render_incidents_table(incidents: List[IncidentReport]) -> Panel:
    """Render the incidents table."""
    table = Table(
        box=box.ROUNDED,
        show_header=True,
        header_style="bold cyan",
        border_style="bright_black",
        expand=True,
        show_lines=True,
    )
    table.add_column("ID", style="bold", min_width=12)
    table.add_column("Severity", min_width=8, justify="center")
    table.add_column("Title", min_width=30)
    table.add_column("Category", min_width=14)
    table.add_column("Status", min_width=14, justify="center")
    table.add_column("Namespaces", min_width=12)
    table.add_column("Confidence", min_width=10, justify="right")
    table.add_column("Detected", min_width=18)

    if not incidents:
        table.add_row(
            "—", "—", "[dim]No incidents found[/dim]", "—", "—", "—", "—", "—"
        )
    else:
        for inc in incidents:
            sev_text = Text(inc.severity.value, style=SEV_STYLE.get(inc.severity, "white"))
            status_text = Text(inc.status.value, style=STATUS_STYLE.get(inc.status, "white"))
            conf_color = "green" if inc.confidence_score >= 80 else ("yellow" if inc.confidence_score >= 50 else "red")
            table.add_row(
                inc.id,
                sev_text,
                Text(inc.title[:50] + ("…" if len(inc.title) > 50 else ""), style="white"),
                Text(inc.category, style="cyan"),
                status_text,
                ", ".join(inc.affected_namespaces[:2]) or "—",
                Text(f"{inc.confidence_score}%", style=conf_color),
                inc.detected_at.strftime("%Y-%m-%d %H:%M"),
            )

    return Panel(
        table,
        title=f"[cyan bold]Active & Recent Incidents ({len(incidents)})[/cyan bold]",
        border_style="cyan",
    )


def render_incident_detail(report: IncidentReport) -> None:
    """Print full detail of a single incident to the console."""
    sev_style = SEV_STYLE.get(report.severity, "white")

    console.print()
    console.print(Rule(f"[bold]Incident {report.id}[/bold]", style="cyan"))
    console.print()

    # Header
    console.print(Panel(
        Text.assemble(
            (f" {report.severity.value} ", sev_style),
            ("  "),
            (report.title, "bold white"),
        ),
        border_style=SEV_COLOR.get(report.severity, "white"),
        padding=(0, 1),
    ))

    # Meta grid
    meta = Table.grid(padding=(0, 2))
    meta.add_column(style="bright_black", min_width=20)
    meta.add_column(style="white")
    meta.add_row("Category", report.category)
    meta.add_row("Status", Text(report.status.value, style=STATUS_STYLE.get(report.status, "white")))
    meta.add_row("Namespaces", ", ".join(report.affected_namespaces) or "—")
    meta.add_row("Confidence", f"{report.confidence_score}%")
    meta.add_row("LLM Model", report.llm_model)
    meta.add_row("Scan Duration", f"{report.scan_duration_seconds:.1f}s" if report.scan_duration_seconds else "—")
    meta.add_row("Runbook", report.runbook_used or "—")
    if report.mttr_seconds:
        meta.add_row("MTTR", f"{report.mttr_seconds // 60}m {report.mttr_seconds % 60}s")
    console.print(Panel(meta, title="Details", border_style="bright_black"))

    # Root cause
    console.print(Panel(
        report.root_cause or "[dim]Not determined[/dim]",
        title="[yellow]Root Cause[/yellow]",
        border_style="yellow",
    ))

    # Contributing factors
    if report.contributing_factors:
        factors_text = "\n".join(f"  • {f}" for f in report.contributing_factors)
        console.print(Panel(factors_text, title="Contributing Factors", border_style="bright_black"))

    # Evidence
    if report.evidence:
        ev_table = Table(box=box.SIMPLE, show_header=True, header_style="bold")
        ev_table.add_column("Type", min_width=10)
        ev_table.add_column("Source", min_width=14)
        ev_table.add_column("Description")
        for ev in report.evidence[:8]:
            ev_table.add_row(ev.type.value, ev.source, ev.description[:80])
        console.print(Panel(ev_table, title="Evidence", border_style="bright_black"))

    # Remediation plan
    if report.remediation_plan:
        console.print()
        console.print("[bold cyan]Remediation Plan:[/bold cyan]")
        for step in report.remediation_plan:
            dest_icon = "🔴" if step.is_destructive else "🟢"
            auto_icon = "🤖" if step.is_automated else "👤"
            console.print(f"  {dest_icon} {auto_icon}  [bold]Step {step.step_number}:[/bold] {step.description}")
            if step.command:
                console.print(f"      [dim]$[/dim] [cyan]{step.command}[/cyan]")
            if step.status != "PENDING":
                console.print(f"      [dim]Status: {step.status}[/dim]")
    console.print()


def render_mttr_stats(stats: dict) -> Panel:
    """Render MTTR statistics table."""
    table = Table(box=box.SIMPLE, show_header=True, header_style="bold cyan")
    table.add_column("Severity", min_width=8)
    table.add_column("Avg MTTR", min_width=12)
    table.add_column("Incidents", min_width=10)

    if not stats:
        table.add_row("—", "No resolved incidents yet", "—")
    else:
        for sev, data in sorted(stats.items()):
            mttr_s = data["avg_mttr_seconds"]
            mttr_str = f"{mttr_s // 60}m {mttr_s % 60}s" if mttr_s else "—"
            sev_enum = Severity[sev] if sev in Severity.__members__ else None
            sev_text = Text(sev, style=SEV_STYLE.get(sev_enum, "white")) if sev_enum else Text(sev)
            table.add_row(sev_text, mttr_str, str(data["count"]))

    return Panel(table, title="[cyan]MTTR Statistics[/cyan]", border_style="bright_black")


def make_scan_progress() -> Progress:
    """Create a progress bar for scan tracking."""
    return Progress(
        SpinnerColumn("dots2", style="cyan"),
        TextColumn("[cyan]{task.description}[/cyan]"),
        TimeElapsedColumn(),
        console=console,
        transient=True,
    )


def prompt_apply_fix(report: IncidentReport) -> bool:
    """
    Interactive prompt to apply the remediation plan.
    Returns True if user approved and commands were run.
    """
    if not report.remediation_plan:
        console.print("[dim]No remediation steps to apply.[/dim]")
        return False

    console.print()
    console.print(Panel(
        "\n".join([
            f"  [bold]{step.step_number}.[/bold] {step.description}"
            + (f"\n     [cyan]$ {step.command}[/cyan]" if step.command else "")
            for step in report.remediation_plan
        ]),
        title="[yellow]Proposed Remediation Steps[/yellow]",
        border_style="yellow",
    ))

    destructive = [s for s in report.remediation_plan if s.is_destructive]
    if destructive:
        console.print(f"[red bold]⚠  {len(destructive)} destructive step(s) in this plan![/red bold]")

    console.print()
    answer = console.input(
        "[bold yellow]Apply this remediation plan? [[green]y[/green]/[red]n[/red]/[cyan]dry[/cyan]]: [/bold yellow]"
    ).strip().lower()

    if answer in ("y", "yes"):
        return True
    elif answer in ("dry", "d"):
        console.print("[cyan]Running in DRY RUN mode — no changes will be made.[/cyan]")
        return False
    else:
        console.print("[dim]Remediation skipped.[/dim]")
        return False
