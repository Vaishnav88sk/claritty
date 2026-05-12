"""
runbook_tools.py — Runbook execution engine for Claritty AI-SRE.

Provides:
  - Runbook YAML loading and listing
  - Safe kubectl command execution with whitelist validation
  - Dry-run mode (default: safe)
  - High-level remediation actions (restart, scale, cordon, delete pod)
"""

import json
import logging
import os
import re
import subprocess
from pathlib import Path
from typing import Optional

import yaml
from crewai.tools import tool

from ..config import config as sre_config

logger = logging.getLogger("claritty.tools.runbook")

RUNBOOKS_DIR = Path(sre_config.runbooks_dir)

# ─── Kubectl command whitelist (safe operations only) ──────
SAFE_KUBECTL_PATTERNS = [
    re.compile(r"^kubectl\s+(get|describe|logs|top)\s+"),
    re.compile(r"^kubectl\s+delete\s+pod\s+[\w\-]+(\s+-n\s+[\w\-]+)?(\s+--force)?(\s+--grace-period=\d+)?$"),
    re.compile(r"^kubectl\s+rollout\s+restart\s+deployment/[\w\-]+(\s+-n\s+[\w\-]+)?$"),
    re.compile(r"^kubectl\s+scale\s+deployment/[\w\-]+\s+--replicas=\d+(\s+-n\s+[\w\-]+)?$"),
    re.compile(r"^kubectl\s+cordon\s+[\w\-\.]+$"),
    re.compile(r"^kubectl\s+uncordon\s+[\w\-\.]+$"),
    re.compile(r"^kubectl\s+annotate\s+"),
    re.compile(r"^kubectl\s+label\s+"),
    re.compile(r"^kubectl\s+set\s+image\s+deployment/[\w\-]+\s+"),
]

DESTRUCTIVE_PATTERNS = [
    re.compile(r"kubectl\s+drain"),
    re.compile(r"kubectl\s+delete\s+(namespace|node|pv|clusterrole)"),
    re.compile(r"kubectl\s+apply\s+-f\s+"),
    re.compile(r"rm\s+-rf"),
]


def _is_safe_command(cmd: str) -> bool:
    cmd = cmd.strip()
    for p in DESTRUCTIVE_PATTERNS:
        if p.search(cmd):
            return False
    for p in SAFE_KUBECTL_PATTERNS:
        if p.match(cmd):
            return True
    return False


def _run_command(cmd: str, dry_run: bool = True) -> dict:
    """Execute a shell command with dry-run guard."""
    if dry_run or sre_config.dry_run:
        return {"dry_run": True, "command": cmd, "output": "[DRY RUN — not executed]"}
    if not _is_safe_command(cmd):
        return {"error": f"Command not in safe whitelist: {cmd}"}
    try:
        result = subprocess.run(
            cmd, shell=True, capture_output=True, text=True, timeout=30
        )
        return {
            "command": cmd,
            "returncode": result.returncode,
            "stdout": result.stdout.strip()[:1000],
            "stderr": result.stderr.strip()[:500],
            "success": result.returncode == 0,
        }
    except subprocess.TimeoutExpired:
        return {"error": "Command timed out after 30s", "command": cmd}
    except Exception as e:
        return {"error": str(e), "command": cmd}


# ──────────────────────────────────────────────────────────
# RUNBOOK TOOLS
# ──────────────────────────────────────────────────────────

@tool
def list_available_runbooks(_input: str = "") -> str:
    """List all available SRE runbooks with their names, trigger conditions,
    and brief descriptions. Use this first to find the right runbook for an issue."""
    if not RUNBOOKS_DIR.exists():
        return json.dumps({"error": f"Runbooks dir not found: {RUNBOOKS_DIR}"})
    runbooks = []
    for f in sorted(RUNBOOKS_DIR.glob("*.yaml")):
        try:
            data = yaml.safe_load(f.read_text())
            runbooks.append({
                "file": f.name,
                "name": data.get("name", f.stem),
                "description": data.get("description", ""),
                "triggers": data.get("triggers", []),
                "severity": data.get("severity", "SEV3"),
                "step_count": len(data.get("steps", [])),
            })
        except Exception as e:
            runbooks.append({"file": f.name, "error": str(e)})
    return json.dumps({"runbooks": runbooks, "count": len(runbooks)})


@tool
def load_runbook(runbook_file: str) -> str:
    """Load and return the full content of a specific runbook YAML file.
    Returns all steps, commands, and descriptions for the runbook.
    Use after list_available_runbooks to get the full remediation plan."""
    path = RUNBOOKS_DIR / runbook_file
    if not path.exists():
        return json.dumps({"error": f"Runbook not found: {runbook_file}"})
    try:
        data = yaml.safe_load(path.read_text())
        return json.dumps(data)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def execute_runbook_step(command: str, dry_run: bool = True) -> str:
    """Execute a single runbook remediation command safely.
    Commands are validated against a whitelist of safe kubectl operations.
    Set dry_run=False only when user has explicitly approved. Default is safe (dry_run=True)."""
    result = _run_command(command, dry_run=dry_run)
    return json.dumps(result)


# ──────────────────────────────────────────────────────────
# HIGH-LEVEL REMEDIATION ACTIONS
# ──────────────────────────────────────────────────────────

@tool
def restart_deployment(deployment_name: str, namespace: str = "default") -> str:
    """Perform a rolling restart of a Kubernetes deployment.
    Safe operation — triggers a new rollout without downtime.
    Use for CrashLoopBackOff pods or after config changes."""
    cmd = f"kubectl rollout restart deployment/{deployment_name} -n {namespace}"
    result = _run_command(cmd)
    result["action"] = "restart_deployment"
    result["target"] = f"{namespace}/{deployment_name}"
    return json.dumps(result)


@tool
def scale_deployment(deployment_name: str, replicas: int,
                     namespace: str = "default") -> str:
    """Scale a Kubernetes deployment to a specified replica count.
    Use to scale UP during high load or scale DOWN to free resources.
    Replicas must be between 0 and 50."""
    replicas = max(0, min(replicas, 50))
    cmd = f"kubectl scale deployment/{deployment_name} --replicas={replicas} -n {namespace}"
    result = _run_command(cmd)
    result["action"] = "scale_deployment"
    result["target"] = f"{namespace}/{deployment_name}"
    result["replicas"] = replicas
    return json.dumps(result)


@tool
def delete_stuck_pod(pod_name: str, namespace: str = "default",
                     force: bool = False) -> str:
    """Delete a stuck or failed pod so its controller recreates it.
    Use for Evicted, Failed, or stuck Terminating pods.
    Set force=True only for pods stuck in Terminating state."""
    force_flags = " --force --grace-period=0" if force else ""
    cmd = f"kubectl delete pod {pod_name} -n {namespace}{force_flags}"
    result = _run_command(cmd)
    result["action"] = "delete_pod"
    result["target"] = f"{namespace}/{pod_name}"
    return json.dumps(result)


@tool
def cordon_node(node_name: str, uncordon: bool = False) -> str:
    """Cordon (or uncordon) a Kubernetes node to prevent new pod scheduling.
    Use cordoning when a node shows disk/memory pressure or needs maintenance.
    Set uncordon=True to re-enable scheduling after the node recovers."""
    action = "uncordon" if uncordon else "cordon"
    cmd = f"kubectl {action} {node_name}"
    result = _run_command(cmd)
    result["action"] = action
    result["node"] = node_name
    return json.dumps(result)


@tool
def describe_cluster_resource(resource_type: str, resource_name: str,
                               namespace: str = "default") -> str:
    """Run kubectl describe on any cluster resource (pod, deployment, node, service, etc.).
    Returns detailed status, events, and conditions. Use for deep diagnosis."""
    safe_types = {"pod", "deployment", "node", "service", "pvc", "event",
                  "replicaset", "statefulset", "daemonset", "job", "cronjob"}
    if resource_type.lower() not in safe_types:
        return json.dumps({"error": f"Resource type '{resource_type}' not in safe list"})
    ns_flag = f" -n {namespace}" if resource_type.lower() not in {"node"} else ""
    cmd = f"kubectl describe {resource_type} {resource_name}{ns_flag}"
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=15)
    return json.dumps({
        "command": cmd,
        "output": result.stdout[:3000],
        "error": result.stderr[:500] if result.returncode != 0 else None,
    })
