# sre_swarm_approved.py
import os
import json
import time
from datetime import datetime
from dotenv import load_dotenv
from crewai import Agent, Task, Crew, LLM
from crewai.tools import tool
from kubernetes import client, config

load_dotenv()
os.environ["GROQ_API_KEY"] = os.getenv("GROQ_API_KEY")

try:
    config.load_kube_config()
except Exception as e:
    print(f"Kube config error: {e}")
    exit(1)

v1 = client.CoreV1Api()

@tool
def get_recent_events(namespace="default", limit=10):
    """Get recent Kubernetes events."""
    try:
        events = v1.list_namespaced_event(namespace, limit=limit)
        return "\n".join([f"{e.last_timestamp} {e.type} {e.reason}: {e.message}"
                         for e in events.items]) or "No events found."
    except Exception as e:
        return f"Error: {e}"

@tool
def list_pods(namespace="default"):
    """List all pods and their status."""
    try:
        pods = v1.list_namespaced_pod(namespace)
        return "\n".join([f"{p.metadata.name}: {p.status.phase}"
                         for p in pods.items]) or "No pods found."
    except Exception as e:
        return f"Error: {e}"

llm = LLM(model="groq/llama-3.3-70b-versatile", temperature=0.2)

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
    allow_delegation=False
)

task_monitor = Task(
    description="Use tools to get recent events and list all pods in default namespace.",
    expected_output="Raw data summary",
    agent=monitor
)

task_detect = Task(
    description=(
        "Analyze monitor data. Look for ImagePullBackOff, CrashLoopBackOff, Pending, Evicted, OOMKilled. "
        "Suggest ONLY safe fixes like 'kubectl delete pod <name> -n default --force --grace-period=0'. "
        "Use 'none' if no specific fix is clear. "
        "Output **only JSON**: "
        "{\"has_issue\": true/false, \"issue\": \"short desc or none\", \"severity\": \"low/medium/high/none\", "
        "\"suggested_fix\": \"full command or none\", \"confidence\": 0-100}"
    ),
    expected_output="Strict JSON",
    agent=detector,
    context=[task_monitor]
)

crew = Crew(agents=[monitor, detector], tasks=[task_monitor, task_detect], verbose=True)

print(f"SRE Swarm Scan - {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
crew_result = crew.kickoff()

raw_output = crew_result.raw if hasattr(crew_result, 'raw') else str(crew_result)
print("\n=== RAW DETECTOR OUTPUT ===\n", raw_output)

try:
    proposal = json.loads(raw_output.strip())
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
    
    try:
        proposal = json.loads(raw_output.strip())
        print("Parsed proposal:", json.dumps(proposal, indent=2))
        
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

# === Automatic loop ===
print("Starting continuous SRE watcher (Ctrl+C to stop)...")
try:
    while True:
        run_scan()
        print("\nWaiting 5 minutes for next scan...\n")
        time.sleep(300)  # 300 seconds = 5 minutes
except KeyboardInterrupt:
    print("\nStopped by user. Goodbye!")