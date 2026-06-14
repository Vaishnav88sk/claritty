"""
log_tools.py — Log analysis tools for Claritty AI-SRE.

Provides error pattern mining, stack trace extraction,
and Loki LogQL support (falls back to kubectl logs).
"""

import json
import logging
import re
import concurrent.futures
from typing import List, Dict, Optional

import requests
from crewai.tools import tool
from kubernetes import client, config as kube_config

from ..config import config as sre_config

logger = logging.getLogger("claritty.tools.logs")

# ─── Common error patterns ────────────────────────────────
ERROR_PATTERNS = [
    (re.compile(r"(FATAL|CRITICAL|PANIC)", re.I), "fatal"),
    (re.compile(r"\b(ERROR|Exception|Traceback|panic:)", re.I), "error"),
    (re.compile(r"(OOMKilled|OutOfMemory|out of memory)", re.I), "oom"),
    (re.compile(r"(connection refused|timeout|ECONNREFUSED|ETIMEDOUT)", re.I), "connectivity"),
    (re.compile(r"(permission denied|unauthorized|forbidden|401|403)", re.I), "auth"),
    (re.compile(r"(CrashLoopBackOff|BackOff)", re.I), "crashloop"),
    (re.compile(r"(disk full|no space left|ENOSPC)", re.I), "disk"),
    (re.compile(r"(segfault|signal 11|SIGSEGV)", re.I), "segfault"),
]

STACK_TRACE_START = re.compile(
    r"(Traceback \(most recent call last\)|goroutine \d+ \[|java\.lang\.|panic:)", re.I
)


def _get_v1() -> client.CoreV1Api:
    try:
        kube_config.load_incluster_config()
    except kube_config.ConfigException:
        kube_config.load_kube_config()
    return client.CoreV1Api()


def _fetch_logs_for_pod(v1: client.CoreV1Api, pod_name: str,
                         namespace: str, tail: int = 200,
                         previous: bool = False) -> str:
    try:
        return v1.read_namespaced_pod_log(
            name=pod_name, namespace=namespace,
            tail_lines=tail, timestamps=True, previous=previous,
        ) or ""
    except Exception as e:
        return f"[log error: {e}]"


def _classify_log_line(line: str) -> Optional[str]:
    for pattern, category in ERROR_PATTERNS:
        if pattern.search(line):
            return category
    return None


def _extract_errors(log_text: str, max_lines: int = 30) -> List[Dict]:
    """Extract error lines with their categories from log text."""
    errors = []
    lines = log_text.split("\n")
    for i, line in enumerate(lines):
        cat = _classify_log_line(line)
        if cat:
            errors.append({
                "line_number": i + 1,
                "category": cat,
                "text": line.strip()[:300],
            })
            if len(errors) >= max_lines:
                break
    return errors


def _extract_stack_traces(log_text: str, max_traces: int = 3) -> List[str]:
    """Extract stack traces from logs (Python, Go, Java)."""
    traces = []
    lines = log_text.split("\n")
    in_trace = False
    current = []

    for line in lines:
        if STACK_TRACE_START.search(line):
            if current and in_trace:
                traces.append("\n".join(current[:30]))
            current = [line]
            in_trace = True
        elif in_trace:
            stripped = line.strip()
            if stripped and (stripped.startswith((" ", "\t", "at ", "File "))
                             or re.match(r"^\w+[\w\.]+\(", stripped)):
                current.append(line)
            elif not stripped:
                current.append(line)
            else:
                if len(current) > 2:
                    traces.append("\n".join(current[:30]))
                current = []
                in_trace = False

    if current and in_trace and len(current) > 2:
        traces.append("\n".join(current[:30]))

    return traces[:max_traces]


# ──────────────────────────────────────────────────────────
# CREW AI TOOLS
# ──────────────────────────────────────────────────────────

@tool
def analyze_pod_logs(pod_name: str, namespace: str = "default",
                     tail_lines: int = 150) -> str:
    """
    Fetch and analyze logs from a pod. Extracts error lines, classifies them
    (fatal/error/oom/connectivity/auth/crashloop/disk/segfault),
    and pulls out stack traces. Also checks previous container logs for crash context.
    Returns structured JSON with error summary and top issues.
    """
    try:
        v1 = _get_v1()
        current_logs = _fetch_logs_for_pod(v1, pod_name, namespace, tail=tail_lines)
        prev_logs = _fetch_logs_for_pod(v1, pod_name, namespace, tail=50, previous=True)

        current_errors = _extract_errors(current_logs)
        prev_errors = _extract_errors(prev_logs)
        stack_traces = _extract_stack_traces(current_logs) or _extract_stack_traces(prev_logs)

        # Category frequency
        cat_freq: Dict[str, int] = {}
        for err in current_errors + prev_errors:
            cat_freq[err["category"]] = cat_freq.get(err["category"], 0) + 1

        return json.dumps({
            "pod": pod_name,
            "namespace": namespace,
            "total_error_lines": len(current_errors),
            "previous_crash_errors": len(prev_errors),
            "error_categories": cat_freq,
            "top_errors": current_errors[:15],
            "previous_errors": prev_errors[:5],
            "stack_traces": stack_traces,
            "raw_tail": current_logs[-2000:] if len(current_logs) > 2000 else current_logs,
        })
    except Exception as e:
        return json.dumps({"error": str(e), "pod": pod_name})


@tool
def analyze_logs_for_namespace(namespace: str = "default",
                                max_pods: int = 10) -> str:
    """
    Analyze logs from all pods in a namespace in parallel.
    Returns aggregated error counts, top error categories, and worst offenders.
    Use when you need a namespace-wide log health picture quickly.
    """
    try:
        v1 = _get_v1()
        pods = v1.list_namespaced_pod(namespace)
        pod_names = [p.metadata.name for p in pods.items
                     if p.status.phase == "Running"][:max_pods]

        def analyze_one(name: str) -> dict:
            logs = _fetch_logs_for_pod(v1, name, namespace, tail=100)
            errors = _extract_errors(logs, max_lines=10)
            cat_freq: Dict[str, int] = {}
            for e in errors:
                cat_freq[e["category"]] = cat_freq.get(e["category"], 0) + 1
            return {"pod": name, "error_count": len(errors), "categories": cat_freq,
                    "top_error": errors[0]["text"] if errors else None}

        results = []
        with concurrent.futures.ThreadPoolExecutor(max_workers=5) as ex:
            futures = {ex.submit(analyze_one, name): name for name in pod_names}
            for fut in concurrent.futures.as_completed(futures):
                try:
                    results.append(fut.result())
                except Exception as e:
                    results.append({"pod": futures[fut], "error": str(e)})

        results.sort(key=lambda x: x.get("error_count", 0), reverse=True)

        total_errors = sum(r.get("error_count", 0) for r in results)
        all_cats: Dict[str, int] = {}
        for r in results:
            for cat, cnt in r.get("categories", {}).items():
                all_cats[cat] = all_cats.get(cat, 0) + cnt

        return json.dumps({
            "namespace": namespace,
            "pods_analyzed": len(results),
            "total_error_lines": total_errors,
            "error_categories": all_cats,
            "pods": results,
        })
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def search_logs_pattern(pod_name: str, pattern: str,
                         namespace: str = "default", tail: int = 500) -> str:
    """
    Search pod logs for a specific regex pattern.
    Useful for finding specific error codes, request IDs, or custom patterns.
    Returns matching lines with line numbers.
    """
    try:
        v1 = _get_v1()
        logs = _fetch_logs_for_pod(v1, pod_name, namespace, tail=tail)
        try:
            regex = re.compile(pattern, re.I)
        except re.error:
            return json.dumps({"error": f"Invalid regex: {pattern}"})

        matches = []
        for i, line in enumerate(logs.split("\n"), 1):
            if regex.search(line):
                matches.append({"line": i, "text": line.strip()[:300]})
        return json.dumps({
            "pod": pod_name, "pattern": pattern,
            "match_count": len(matches), "matches": matches[:30],
        })
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def loki_query(logql: str, limit: int = 50) -> str:
    """
    Execute a LogQL query against Loki for log aggregation.
    Example: '{namespace="default", app="nginx"} |= "error"'
    Falls back gracefully if Loki is not available.
    Use for structured log queries across multiple pods.
    """
    if not sre_config.loki_enabled:
        return json.dumps({"status": "disabled",
                           "message": "Loki not enabled. Set LOKI_ENABLED=true in .env"})
    try:
        resp = requests.get(
            f"{sre_config.loki_url}/loki/api/v1/query_range",
            params={"query": logql, "limit": limit, "direction": "backward"},
            timeout=15,
        )
        resp.raise_for_status()
        data = resp.json()
        streams = data.get("data", {}).get("result", [])
        results = []
        for stream in streams:
            labels = stream.get("stream", {})
            for ts, line in stream.get("values", []):
                results.append({"labels": labels, "timestamp": ts, "log": line[:300]})
        return json.dumps({"status": "ok", "count": len(results), "logs": results[:50]})
    except Exception as e:
        return json.dumps({"status": "error", "error": str(e)})
