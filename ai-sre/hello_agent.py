# hello_agent.py
import os
from datetime import datetime
from dotenv import load_dotenv
from crewai import Agent, Task, Crew, LLM
from crewai.tools import tool           # ← correct import

from kubernetes import client, config

load_dotenv()
os.environ["GROQ_API_KEY"] = os.getenv("GROQ_API_KEY")

# Load Kubernetes config
try:
    config.load_kube_config()
except Exception as e:
    print(f"Warning: Cannot load kubeconfig → {e}")
    print("Run 'minikube start' first")

v1 = client.CoreV1Api()

# === This is the best way: use @tool decorator ===
@tool
def kubernetes_recent_events(namespace: str = "default", limit: int = 10) -> str:
    """Get the most recent Kubernetes events from a namespace.
    Useful for checking warnings, errors, restarts, evictions."""
    try:
        events = v1.list_namespaced_event(namespace, limit=limit)
        lines = []
        for ev in events.items:
            ts = ev.last_timestamp or ev.event_time or "unknown"
            lines.append(f"{ts} {ev.type} {ev.reason}: {ev.message}")
        return "\n".join(lines) or "No events found."
    except Exception as e:
        return f"Error reading events: {str(e)}"

llm = LLM(model="groq/llama-3.3-70b-versatile", temperature=0.7)

agent = Agent(
    role="Kubernetes Observer",
    goal="Monitor cluster and answer questions",
    backstory="You are a junior SRE who watches Kubernetes clusters.",
    llm=llm,
    verbose=True,
    allow_delegation=False,
    tools=[kubernetes_recent_events]          # ← pass the decorated function
)

task = Task(
    description=(
        "1. Use the kubernetes_recent_events tool to get the 10 most recent events "
        "in namespace 'default'.\n"
        "2. Look at the events and tell me if you see any warnings, errors, or unusual patterns.\n"
        "3. Finally explain in 2-3 simple sentences what a Pod Disruption Budget (PDB) is."
    ),
    expected_output="Events + short analysis + PDB explanation",
    agent=agent
)

crew = Crew(agents=[agent], tasks=[task], verbose=True)

print(f"Starting at {datetime.now().strftime('%H:%M:%S')}\n")
result = crew.kickoff()
print("\nFinal result:\n")
print(result)