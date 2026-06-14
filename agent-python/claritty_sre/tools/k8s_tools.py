"""
k8s_tools.py — CrewAI-compatible Kubernetes tools for Claritty AI-SRE.

Wraps the kubernetes Python client to provide rich, structured data
for agent analysis. All tools return JSON strings for LLM consumption.
"""

import json
import logging
from typing import Optional, List

from crewai.tools import tool
from kubernetes import client, config as kube_config

logger = logging.getLogger("claritty.tools.k8s")

# ─── K8s Client Init (lazy, cached) ───────────────────────
_v1: Optional[client.CoreV1Api] = None
_apps_v1: Optional[client.AppsV1Api] = None
_autoscaling_v1: Optional[client.AutoscalingV1Api] = None

def _get_v1() -> client.CoreV1Api:
    global _v1
    if _v1 is None:
        try:
            kube_config.load_incluster_config()
        except kube_config.ConfigException:
            kube_config.load_kube_config()
        _v1 = client.CoreV1Api()
    return _v1

def _get_apps() -> client.AppsV1Api:
    global _apps_v1
    if _apps_v1 is None:
        _get_v1()  # ensures config loaded
        _apps_v1 = client.AppsV1Api()
    return _apps_v1

def _get_autoscaling() -> client.AutoscalingV1Api:
    global _autoscaling_v1
    if _autoscaling_v1 is None:
        _get_v1()
        _autoscaling_v1 = client.AutoscalingV1Api()
    return _autoscaling_v1


# ──────────────────────────────────────────────────────────
# PODS
# ──────────────────────────────────────────────────────────

@tool
def list_pods_all_namespaces(query: str = "") -> str:
    """
    List all pods across all namespaces with their status and restart counts.
    Focuses on non-running pods and problem indicators.
    """
    try:
        v1 = _get_v1()
        pods = v1.list_pod_for_all_namespaces()
        result = []
        for p in pods.items:
            # Skip healthy kube-system pods to save tokens
            if p.metadata.namespace == "kube-system" and p.status.phase == "Running":
                continue

            restart_count = 0
            states = []
            if p.status.container_statuses:
                for cs in p.status.container_statuses:
                    restart_count += cs.restart_count
                    state = "running"
                    if cs.state.waiting:
                        state = f"waiting:{cs.state.waiting.reason}"
                    elif cs.state.terminated:
                        state = f"exit:{cs.state.terminated.exit_code}"
                    states.append({"name": cs.name, "state": state, "restarts": cs.restart_count})

            result.append({
                "ns": p.metadata.namespace,
                "pod": p.metadata.name,
                "phase": p.status.phase,
                "restarts": restart_count,
                "states": states,
            })
        return json.dumps(result, default=str)
    except Exception as e:
        logger.error("list_pods_all_namespaces: %s", e)
        return json.dumps({"error": str(e)})


@tool
def describe_pod(pod_name: str, namespace: str = "default") -> str:
    """
    Get a full description of a specific pod: events, conditions, init containers,
    volumes, resource usage, and previous container logs (for crashloop diagnosis).
    Use this when you need deep detail about a failing pod.
    """
    try:
        v1 = _get_v1()
        pod = v1.read_namespaced_pod(pod_name, namespace)

        # Events for this pod
        events = v1.list_namespaced_event(namespace)
        pod_events = [
            {
                "time": str(e.last_timestamp),
                "type": e.type,
                "reason": e.reason,
                "message": e.message,
                "count": e.count,
            }
            for e in events.items
            if e.involved_object.name == pod_name
        ]

        # Init container states
        init_states = []
        if pod.status.init_container_statuses:
            for ics in pod.status.init_container_statuses:
                state = "unknown"
                if ics.state.running:
                    state = "running"
                elif ics.state.waiting:
                    state = f"waiting:{ics.state.waiting.reason}"
                elif ics.state.terminated:
                    state = f"terminated:{ics.state.terminated.reason}:{ics.state.terminated.exit_code}"
                init_states.append({"name": ics.name, "state": state, "restarts": ics.restart_count})

        # Main container states + previous logs hint
        container_detail = []
        if pod.status.container_statuses:
            for cs in pod.status.container_statuses:
                detail = {
                    "name": cs.name,
                    "ready": cs.ready,
                    "restart_count": cs.restart_count,
                    "image": cs.image,
                }
                if cs.state.waiting:
                    detail["waiting_reason"] = cs.state.waiting.reason
                    detail["waiting_message"] = cs.state.waiting.message
                if cs.last_state.terminated:
                    t = cs.last_state.terminated
                    detail["last_exit_code"] = t.exit_code
                    detail["last_reason"] = t.reason
                container_detail.append(detail)

        return json.dumps({
            "name": pod_name,
            "namespace": namespace,
            "phase": pod.status.phase,
            "node": pod.spec.node_name,
            "start_time": str(pod.status.start_time),
            "conditions": [
                {"type": c.type, "status": c.status, "reason": c.reason, "message": c.message}
                for c in (pod.status.conditions or [])
            ],
            "init_containers": init_states,
            "containers": container_detail,
            "events": pod_events[-10:],  # Last 10 events
        }, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def get_pod_logs(pod_name: str, namespace: str = "default",
                 container: str = "", tail_lines: int = 100,
                 previous: bool = False) -> str:
    """
    Fetch the last N lines of logs from a pod's container.
    Set previous=True to get logs from a previously crashed container (useful for CrashLoopBackOff).
    Set container name if the pod has multiple containers.
    """
    try:
        v1 = _get_v1()
        kwargs = dict(
            name=pod_name,
            namespace=namespace,
            tail_lines=tail_lines,
            timestamps=True,
            previous=previous,
        )
        if container:
            kwargs["container"] = container
        logs = v1.read_namespaced_pod_log(**kwargs)
        return logs or "No logs available."
    except Exception as e:
        return f"Error getting logs for {pod_name}: {str(e)}"


# ──────────────────────────────────────────────────────────
# EVENTS
# ──────────────────────────────────────────────────────────

@tool
def get_cluster_events(namespace: str = "", warning_only: bool = True,
                       limit: int = 20) -> str:
    """
    Get recent Kubernetes Warning events across the cluster.
    Sorted by most recent first. Use for initial triage.
    """
    try:
        v1 = _get_v1()
        if namespace:
            events = v1.list_namespaced_event(namespace, limit=100)
        else:
            events = v1.list_event_for_all_namespaces(limit=100)

        items = events.items
        if warning_only:
            items = [e for e in items if e.type == "Warning"]

        items.sort(
            key=lambda e: e.last_timestamp or e.event_time or "",
            reverse=True,
        )

        # Deduplicate by (reason, object) — keep only latest of each
        seen = set()
        deduped = []
        for e in items:
            key = (e.reason, e.involved_object.name)
            if key not in seen:
                seen.add(key)
                deduped.append(e)

        return json.dumps([
            {
                "reason": e.reason,
                "object": f"{e.involved_object.kind}/{e.involved_object.name}",
                "msg": (e.message or "")[:120],  # truncate long messages
                "count": e.count,
            }
            for e in deduped[:limit]
        ], default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


# ──────────────────────────────────────────────────────────
# NODES
# ──────────────────────────────────────────────────────────

@tool
def get_node_health(query: str = "") -> str:
    """
    Get detailed health status of all cluster nodes including:
    Ready/NotReady status, memory/disk/PID pressure, taints,
    and allocatable vs capacity resources. Essential for infrastructure issues.
    """
    try:
        v1 = _get_v1()
        nodes = v1.list_node()
        result = []
        for n in nodes.items:
            conditions = {c.type: {"status": c.status, "reason": c.reason, "message": c.message}
                         for c in n.status.conditions}
            taints = [
                {"key": t.key, "effect": t.effect, "value": t.value}
                for t in (n.spec.taints or [])
            ]
            result.append({
                "name": n.metadata.name,
                "ready": conditions.get("Ready", {}).get("status") == "True",
                "conditions": conditions,
                "taints": taints,
                "capacity": {
                    "cpu_cores": str(n.status.capacity.get("cpu", "—")),
                    "memory_mb": int(n.status.capacity.get("memory", "0").replace("Ki", "")) // 1024
                    if "Ki" in str(n.status.capacity.get("memory", "")) else "—",
                    "pods": str(n.status.capacity.get("pods", "—")),
                },
                "allocatable": {
                    "cpu_cores": str(n.status.allocatable.get("cpu", "—")),
                    "memory_mb": int(n.status.allocatable.get("memory", "0").replace("Ki", "")) // 1024
                    if "Ki" in str(n.status.allocatable.get("memory", "")) else "—",
                },
                "labels": {k: v for k, v in n.metadata.labels.items()
                           if k.startswith("node.kubernetes.io") or k == "kubernetes.io/hostname"},
                "os": n.status.node_info.os_image,
                "kernel": n.status.node_info.kernel_version,
            })
        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


# ──────────────────────────────────────────────────────────
# DEPLOYMENTS / WORKLOADS
# ──────────────────────────────────────────────────────────

@tool
def get_deployments_status(namespace: str = "") -> str:
    """
    List all deployments and their replica status (desired vs ready vs available).
    Identifies replica mismatches indicating rollout failures or pod crashes.
    """
    try:
        apps = _get_apps()
        if namespace:
            deploys = apps.list_namespaced_deployment(namespace)
        else:
            deploys = apps.list_deployment_for_all_namespaces()

        result = []
        for d in deploys.items:
            spec_replicas = d.spec.replicas or 0
            status_ready = d.status.ready_replicas or 0
            status_available = d.status.available_replicas or 0
            status_updated = d.status.updated_replicas or 0

            result.append({
                "namespace": d.metadata.namespace,
                "name": d.metadata.name,
                "desired": spec_replicas,
                "ready": status_ready,
                "available": status_available,
                "updated": status_updated,
                "is_healthy": status_ready == spec_replicas and spec_replicas > 0,
                "conditions": [
                    {"type": c.type, "status": c.status, "reason": c.reason, "message": c.message}
                    for c in (d.status.conditions or [])
                ],
                "strategy": d.spec.strategy.type,
                "image": d.spec.template.spec.containers[0].image
                if d.spec.template.spec.containers else "—",
            })
        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def get_resource_quotas(namespace: str = "default") -> str:
    """
    Get resource quotas and limits for a namespace.
    Essential for diagnosing Pending pods due to quota exhaustion.
    """
    try:
        v1 = _get_v1()
        quotas = v1.list_namespaced_resource_quota(namespace)
        limit_ranges = v1.list_namespaced_limit_range(namespace)

        result = {
            "namespace": namespace,
            "quotas": [],
            "limit_ranges": [],
        }

        for q in quotas.items:
            result["quotas"].append({
                "name": q.metadata.name,
                "hard": dict(q.status.hard or {}),
                "used": dict(q.status.used or {}),
            })

        for lr in limit_ranges.items:
            for lim in (lr.spec.limits or []):
                result["limit_ranges"].append({
                    "type": lim.type,
                    "default": dict(lim.default or {}),
                    "default_request": dict(lim.default_request or {}),
                    "max": dict(lim.max or {}),
                    "min": dict(lim.min or {}),
                })

        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def get_pvc_status(namespace: str = "") -> str:
    """
    Get PersistentVolumeClaim status. Identifies unbound PVCs that block pod scheduling,
    full volumes causing disk pressure, or storage class mismatches.
    """
    try:
        v1 = _get_v1()
        if namespace:
            pvcs = v1.list_namespaced_persistent_volume_claim(namespace)
        else:
            pvcs = v1.list_persistent_volume_claim_for_all_namespaces()

        result = []
        for pvc in pvcs.items:
            result.append({
                "namespace": pvc.metadata.namespace,
                "name": pvc.metadata.name,
                "phase": pvc.status.phase,
                "storage_class": pvc.spec.storage_class_name,
                "capacity": dict(pvc.status.capacity or {}),
                "access_modes": pvc.spec.access_modes,
                "volume_name": pvc.spec.volume_name,
                "is_bound": pvc.status.phase == "Bound",
            })
        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def get_hpa_status(namespace: str = "") -> str:
    """
    Get HorizontalPodAutoscaler status: current vs desired replicas,
    min/max bounds, and current metric values. Helps diagnose scaling issues.
    """
    try:
        autoscaling = _get_autoscaling()
        if namespace:
            hpas = autoscaling.list_namespaced_horizontal_pod_autoscaler(namespace)
        else:
            hpas = autoscaling.list_horizontal_pod_autoscaler_for_all_namespaces()

        result = []
        for h in hpas.items:
            result.append({
                "namespace": h.metadata.namespace,
                "name": h.metadata.name,
                "target": h.spec.scale_target_ref.name,
                "min_replicas": h.spec.min_replicas,
                "max_replicas": h.spec.max_replicas,
                "current_replicas": h.status.current_replicas,
                "desired_replicas": h.status.desired_replicas,
                "current_cpu_pct": h.status.current_cpu_utilization_percentage,
                "target_cpu_pct": h.spec.target_cpu_utilization_percentage,
                "at_max": h.status.current_replicas >= h.spec.max_replicas,
            })
        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})


@tool
def get_namespace_summary(query: str = "") -> str:
    """
    Get a summary of all namespaces: pod counts by phase, resource usage,
    and namespace-level health indicators. Good for multi-tenant cluster analysis.
    """
    try:
        v1 = _get_v1()
        namespaces = v1.list_namespace()
        pods_all = v1.list_pod_for_all_namespaces()

        # Group pods by namespace
        ns_pods: dict = {}
        for p in pods_all.items:
            ns = p.metadata.namespace
            if ns not in ns_pods:
                ns_pods[ns] = {"running": 0, "pending": 0, "failed": 0, "total": 0, "crashloop": 0}
            ns_pods[ns]["total"] += 1
            phase = p.status.phase
            if phase == "Running":
                ns_pods[ns]["running"] += 1
            elif phase == "Pending":
                ns_pods[ns]["pending"] += 1
            elif phase in ("Failed", "Unknown"):
                ns_pods[ns]["failed"] += 1
            # Check for crashloop
            if p.status.container_statuses:
                for cs in p.status.container_statuses:
                    if cs.state.waiting and cs.state.waiting.reason == "CrashLoopBackOff":
                        ns_pods[ns]["crashloop"] += 1

        result = []
        for ns in namespaces.items:
            name = ns.metadata.name
            result.append({
                "namespace": name,
                "status": ns.status.phase,
                "pods": ns_pods.get(name, {"running": 0, "pending": 0, "failed": 0, "total": 0, "crashloop": 0}),
            })

        return json.dumps(result, default=str)
    except Exception as e:
        return json.dumps({"error": str(e)})
