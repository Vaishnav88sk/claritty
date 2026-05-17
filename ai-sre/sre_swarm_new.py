# sre_swarm_approved.py
import os
import json
import time
import re
import subprocess
from datetime import datetime
from dotenv import load_dotenv
from crewai import Agent, Task, Crew, LLM
from crewai.tools import tool
from kubernetes import client, config
from pydantic import BaseModel

class OOMInput(BaseModel):
    pass


load_dotenv()
os.environ["GROQ_API_KEY"] = os.getenv("GROQ_API_KEY")
os.environ["MISTRAL_API_KEY"] = os.getenv("MISTRAL_API_KEY")

try:
    config.load_kube_config()
except Exception as e:
    print(f"Kube config error: {e}")
    exit(1)

v1 = client.CoreV1Api()

def clean_json_output(raw: str) -> str:
    return re.sub(r"```json|```", "", raw).strip()

def pod_exists(pod_name):
    result = subprocess.run(
        ["kubectl", "get", "pod", pod_name, "-n", "default"],
        capture_output=True,
        text=True
    )
    return result.returncode == 0

@tool
def get_recent_events(namespace="default", limit=10):
    """Get recent Kubernetes events."""
    try:
        events = v1.list_namespaced_event(namespace, limit=limit)
        return json.dumps([
            {
                "time": str(e.last_timestamp),
                "type": e.type,
                "reason": e.reason,
                "message": e.message
            }
            for e in events.items
        ])
    except Exception as e:
        return f"Error: {e}"

@tool
def list_pods(namespace="default"):
    """List all pods and their status."""
    try:
        pods = v1.list_namespaced_pod(namespace)
        return json.dumps([
            {"name": p.metadata.name, "status": p.status.phase}
            for p in pods.items
        ])
    except Exception as e:
        return f"Error: {e}"

@tool
def get_pod_logs(pod_name: str, namespace="default", tail_lines=50):
    """Get the last N log lines from a specific pod to understand why it failed."""
    try:
        logs = v1.read_namespaced_pod_log(
            name=pod_name,
            namespace=namespace,
            tail_lines=tail_lines,
            timestamps=True
        )
        return logs or "No logs available."
    except Exception as e:
        return f"Error getting logs: {str(e)}"


@tool
def get_prometheus_metrics(query="container_cpu_usage_seconds_total", limit=10):
    """Get Prometheus metrics for CPU, memory, etc. Use for real SRE issues like high CPU or memory pressure."""
    from prometheus_client import CollectorRegistry, query
    try:
        registry = CollectorRegistry()
        result = query.prometheus_query(query, registry)  # assume Prometheus is installed
        return str(result) or "No metrics found."
    except Exception as e:
        return f"Error: {e}"

@tool()
def get_oom_traces(duration: int = 5):
    """Check for recent OOM kills using bpftrace (requires sudo, run agent as root or with privileges)."""
    try:
        import subprocess
        cmd = f"sudo bpftrace -e 'kprobe:__oom_kill_process {{ printf(\"%s killed %d\\n\", comm, pid); }}' -c 'sleep {duration}'"
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=10)
        return result.stdout.strip() or "No recent OOM kills detected."
    except Exception as e:
        return f"Error: {str(e)} (bpftrace may need sudo)"

# llm = LLM(model="groq/llama-3.3-70b-versatile", temperature=0.2)
# llm = LLM(model="groq/gemma2-9b-it", temperature=0.2)
llm = LLM(model="mistral/mistral-large-latest", temperature=0.2, max_tokens=1024)

monitor = Agent(
    role="Monitor",
    goal="Collect cluster data",
    backstory="You gather events and pod status.",
    llm=llm,
    tools=[get_recent_events, list_pods],
    verbose=True,
    allow_delegation=False
)

detector = Agent(
    role="Detector",
    goal="Find issues and suggest safe fixes",
    backstory="You are careful. Suggest only safe commands like delete pod. Use 'none' if unsure.",
    llm=llm,
    verbose=True,
    allow_delegation=False,
    tools=[get_pod_logs, get_oom_traces]
)

task_monitor = Task(
    description="Use tools to get recent events and list all pods in default namespace.",
    expected_output="Raw data summary",
    agent=monitor
)

task_detect = Task(
    description=(
        "Analyze ONLY the data provided from the monitor task.\n"
        "\n"
        "STRICT RULES:\n"
        "- Do NOT assume anything.\n"
        "- Do NOT hallucinate pods, events, or metrics.\n"
        "- Only report issues that are explicitly present in the data.\n"
        "\n"
        "IMPORTANT:\n"
        "- Kubernetes events may refer to OLD or deleted pods.\n"
        "- If a pod is NOT present in the current pod list, DO NOT report it as an active issue.\n"
        "- Only consider issues for pods that currently exist.\n"
        "\n"
        "If NO active issues are found, return EXACTLY:\n"
        "{\"has_issue\": false, \"issue\": \"none\", \"severity\": \"none\", \"suggested_fix\": \"none\", \"confidence\": 100}\n"
        "\n"
        "If issues ARE found:\n"
        "- Base them ONLY on CURRENT pods\n"
        "- You may use tools if needed\n"
        "- Suggest ONLY safe fixes\n"
        "\n"
        "OUTPUT RULES:\n"
        "- Output ONLY valid JSON\n"
        "- No markdown, no explanation\n"
    ),
    expected_output="Strict JSON",
    agent=detector,
    context=[task_monitor]
)

crew = Crew(agents=[monitor, detector], tasks=[task_monitor, task_detect], verbose=True, tracing=False)

print(f"SRE Swarm Scan - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
crew_result = crew.kickoff()

raw_output = crew_result.raw if hasattr(crew_result, 'raw') else str(crew_result)
print("\n=== RAW DETECTOR OUTPUT ===\n", raw_output)

try:
    cleaned = clean_json_output(raw_output)
    proposal = json.loads(cleaned)
    print("\nParsed proposal:", json.dumps(proposal, indent=2))

    if proposal.get("has_issue", False):
        fix = proposal.get("suggested_fix", "none")
        if fix != "none":
            print(f"\nProposed fix: {fix}")
            print(f"Confidence: {proposal['confidence']}% | Severity: {proposal['severity']}")
            answer = input("\nApply this fix? (y/n): ").strip().lower()
            if answer == 'y':
                print("Applying...")
                os.system(fix)
                print("Fix applied!")
            else:
                print("Skipped.")
        else:
            print("No fix suggested.")
    else:
        print("No issues - cluster healthy!")
except Exception as e:
    print("Parse error:", e)
    print("Raw:", raw_output)


def run_scan():
    print(f"\n=== New Scan at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')} ===\n")
    crew_result = crew.kickoff()
    raw_output = crew_result.raw if hasattr(crew_result, 'raw') else str(crew_result)
    if "No pods found" in raw_output or raw_output.strip() == "":
        print("No issues - cluster empty.")
        return
    
    try:
        cleaned = clean_json_output(raw_output)
        proposal = json.loads(cleaned)
        print("Parsed proposal:", json.dumps(proposal, indent=2))
        
        if proposal.get("has_issue", False):
            fix = proposal.get("suggested_fix", "none")
            if fix != "none":
                print(f"\nProposed fix: {fix}")
                print(f"Confidence: {proposal['confidence']}% | Severity: {proposal['severity']}")
                answer = input("\nApply this fix? (y/n): ").strip().lower()
                if answer == 'y':
                    print("Applying...")
                    if "delete pod" in fix:
                        pod_name = fix.split()[3]

                        if not pod_exists(pod_name):
                            print(f"⚠️ Pod {pod_name} does NOT exist. Skipping fix.")
                        else:
                            os.system(fix)
                    else:
                        os.system(fix)
                    print("Fix applied!")
                else:
                    print("Skipped.")
            else:
                print("No fix suggested.")
        else:
            print("No issues - cluster healthy!")
    except Exception as e:
        print("Parse error:", e)
        print("Raw:", raw_output)

# === Automatic loop ===
print("Starting continuous SRE watcher (Ctrl+C to stop)...")
try:
    while True:
        run_scan()
        print("\nWaiting 5 minutes for next scan...\n")
        time.sleep(300)  # 300 seconds = 5 minutes
except KeyboardInterrupt:
    print("\nStopped by user. Goodbye!")