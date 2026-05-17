"""
metrics_tools.py — Prometheus/metrics tools for Claritty AI-SRE.
Queries Prometheus via HTTP API with z-score anomaly detection.
Falls back gracefully when Prometheus is unavailable.
"""

import json
import logging
import math
import time
from typing import List, Tuple

import requests
from crewai.tools import tool

from ..config import config

logger = logging.getLogger("claritty.tools.metrics")

PROM_URL = config.prometheus_url
WINDOW = config.metrics_window


def _prom_query(query: str) -> dict:
    try:
        resp = requests.get(f"{PROM_URL}/api/v1/query", params={"query": query}, timeout=10)
        resp.raise_for_status()
        return resp.json()
    except requests.exceptions.ConnectionError:
        return {"status": "error", "error": f"Cannot connect to Prometheus at {PROM_URL}"}
    except Exception as e:
        return {"status": "error", "error": str(e)}


def _prom_range(query: str, start: float, end: float, step: str = "60s") -> dict:
    try:
        resp = requests.get(
            f"{PROM_URL}/api/v1/query_range",
            params={"query": query, "start": start, "end": end, "step": step},
            timeout=15,
        )
        resp.raise_for_status()
        return resp.json()
    except Exception as e:
        return {"status": "error", "error": str(e)}


def _results(resp: dict) -> list:
    if resp.get("status") != "success":
        return []
    return resp.get("data", {}).get("result", [])


def _zscore(values: List[float], threshold: float = 2.5) -> Tuple[bool, float]:
    if len(values) < 5:
        return False, 0.0
    mean = sum(values) / len(values)
    std = math.sqrt(sum((v - mean) ** 2 for v in values) / len(values))
    if std == 0:
        return False, 0.0
    z = (values[-1] - mean) / std
    return abs(z) > threshold, round(z, 2)


@tool
def get_cpu_usage_per_pod(namespace: str = "") -> str:
    """Get CPU usage (millicores) per pod over the last 5 minutes using Prometheus PromQL.
    Returns top pods by CPU consumption. Use to identify CPU-hungry pods."""
    ns = f', namespace="{namespace}"' if namespace else ""
    q = f'sort_desc(sum by(pod,namespace)(rate(container_cpu_usage_seconds_total{{container!=""{ns}}}[{WINDOW}])))'
    items = []
    for r in _results(_prom_query(q))[:30]:
        val = float(r["value"][1]) if r.get("value") else 0.0
        items.append({
            "pod": r["metric"].get("pod", "—"),
            "namespace": r["metric"].get("namespace", "—"),
            "cpu_millicores": round(val * 1000, 1),
        })
    return json.dumps({"status": "ok", "window": WINDOW, "pods": items})


@tool
def get_memory_usage_per_pod(namespace: str = "") -> str:
    """Get memory usage (MB) per pod via Prometheus container_memory_working_set_bytes.
    Identifies memory-heavy or leaking pods. Returns top 30 by memory consumption."""
    ns = f', namespace="{namespace}"' if namespace else ""
    q = f'sort_desc(sum by(pod,namespace)(container_memory_working_set_bytes{{container!=""{ns}}}))'
    items = []
    for r in _results(_prom_query(q))[:30]:
        val = float(r["value"][1]) if r.get("value") else 0.0
        items.append({
            "pod": r["metric"].get("pod", "—"),
            "namespace": r["metric"].get("namespace", "—"),
            "memory_mb": round(val / (1024 * 1024), 1),
        })
    return json.dumps({"status": "ok", "pods": items})


@tool
def get_pod_restart_rate(namespace: str = "") -> str:
    """Get pod restart counts in the last 30 minutes.
    High restart rates signal CrashLoopBackOff or OOM kills. Returns pods with >0 restarts."""
    ns = f', namespace="{namespace}"' if namespace else ""
    q = f'sort_desc(increase(kube_pod_container_status_restarts_total{{container!=""{ns}}}[30m]))'
    items = []
    for r in _results(_prom_query(q)):
        val = float(r["value"][1]) if r.get("value") else 0.0
        if val > 0:
            items.append({
                "pod": r["metric"].get("pod", "—"),
                "container": r["metric"].get("container", "—"),
                "namespace": r["metric"].get("namespace", "—"),
                "restarts_30m": round(val, 1),
            })
    return json.dumps({"status": "ok", "pods": items[:20]})


@tool
def get_http_error_rate(namespace: str = "") -> str:
    """Compute HTTP 5xx error rate per service via Prometheus http_requests_total.
    Returns error_rate_pct per service. High values (>1%) indicate service problems."""
    ns = f', namespace="{namespace}"' if namespace else ""
    err_q = f'sum by(service,namespace)(rate(http_requests_total{{code=~"5.."{ns}}}[{WINDOW}]))'
    ok_q  = f'sum by(service,namespace)(rate(http_requests_total{{code!~"5.."{ns}}}[{WINDOW}]))'
    errors = {(r["metric"].get("service",""), r["metric"].get("namespace","")): float(r["value"][1])
              for r in _results(_prom_query(err_q))}
    totals = {(r["metric"].get("service",""), r["metric"].get("namespace","")): float(r["value"][1])
              for r in _results(_prom_query(ok_q))}
    result = []
    for key, err in errors.items():
        tot = totals.get(key, 0)
        pct = err / (tot + err) * 100 if (tot + err) > 0 else 0
        result.append({"service": key[0] or "—", "namespace": key[1] or "—",
                        "error_rate_pct": round(pct, 2), "error_rps": round(err, 4)})
    result.sort(key=lambda x: x["error_rate_pct"], reverse=True)
    return json.dumps({"status": "ok", "window": WINDOW, "services": result[:20]})


@tool
def get_request_latency_p99(namespace: str = "") -> str:
    """Get P99 request latency (ms) per service via Prometheus histogram_quantile.
    High P99 latency means slow responses even if error rate is low. Returns top 20 services."""
    q = ('sort_desc(histogram_quantile(0.99, sum by(service,namespace,le)'
         f'(rate(http_request_duration_seconds_bucket[{WINDOW}]))))')
    items = []
    for r in _results(_prom_query(q))[:20]:
        val = float(r["value"][1]) if r.get("value") else 0.0
        if not math.isnan(val) and not math.isinf(val):
            items.append({
                "service": r["metric"].get("service", "—"),
                "namespace": r["metric"].get("namespace", "—"),
                "p99_latency_ms": round(val * 1000, 1),
            })
    return json.dumps({"status": "ok", "window": WINDOW, "services": items})


@tool
def get_node_resource_pressure(query: str = "") -> str:
    """Get CPU and memory utilization percentages per node from Prometheus node_exporter.
    Identifies overloaded nodes needing scale-out or pod eviction."""
    cpu_q = '100 - (avg by(instance)(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)'
    mem_q = '100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))'
    nodes: dict = {}
    for r in _results(_prom_query(cpu_q)):
        inst = r["metric"].get("instance", r["metric"].get("node", "—"))
        nodes.setdefault(inst, {})["cpu_pct"] = round(float(r["value"][1]), 1)
    for r in _results(_prom_query(mem_q)):
        inst = r["metric"].get("instance", r["metric"].get("node", "—"))
        nodes.setdefault(inst, {})["mem_pct"] = round(float(r["value"][1]), 1)
    result = [{
        "node": k,
        "cpu_pct": v.get("cpu_pct", "—"),
        "mem_pct": v.get("mem_pct", "—"),
        "cpu_critical": isinstance(v.get("cpu_pct"), float) and v["cpu_pct"] > config.cpu_critical_pct,
        "mem_critical": isinstance(v.get("mem_pct"), float) and v["mem_pct"] > config.memory_critical_pct,
    } for k, v in nodes.items()]
    return json.dumps({"status": "ok", "nodes": result})


@tool
def get_oom_kill_events(query: str = "") -> str:
    """Detect recent OOM kill events using kube_pod_container_status_last_terminated_reason.
    OOMKilled pods need higher memory limits. Returns all affected pods."""
    q = 'kube_pod_container_status_last_terminated_reason{reason="OOMKilled"} == 1'
    pods = [{
        "pod": r["metric"].get("pod", "—"),
        "namespace": r["metric"].get("namespace", "—"),
        "container": r["metric"].get("container", "—"),
    } for r in _results(_prom_query(q))]
    return json.dumps({"status": "ok", "oom_count": len(pods), "pods": pods})


@tool
def detect_cpu_anomaly_pod(pod_name: str, namespace: str = "default") -> str:
    """Run z-score anomaly detection on a pod's CPU usage over the last hour.
    z-score > 2.5 means the current CPU is statistically anomalous vs baseline.
    Use this to confirm if a specific pod is abnormally spiking."""
    end = time.time()
    start = end - 3600
    q = f'rate(container_cpu_usage_seconds_total{{pod="{pod_name}",namespace="{namespace}",container!=""}}[2m])'
    data = _prom_range(q, start, end, step="120s")
    items = _results(data)
    if not items:
        return json.dumps({"status": "no_data", "pod": pod_name})
    values = [float(v[1]) for v in items[0].get("values", []) if v[1] != "NaN"]
    if not values:
        return json.dumps({"status": "no_data", "pod": pod_name})
    is_anom, z = _zscore(values)
    return json.dumps({
        "pod": pod_name, "namespace": namespace,
        "is_anomaly": is_anom, "z_score": z,
        "current_cpu_millicores": round(values[-1] * 1000, 1),
        "avg_cpu_millicores": round(sum(values) / len(values) * 1000, 1),
    })
