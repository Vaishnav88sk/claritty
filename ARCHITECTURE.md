# Claritty Architecture & Repository Structure

Claritty is a decentralized, cloud-native AI Site Reliability Engineering (AI-SRE) platform for Kubernetes. It combines real-time cluster telemetry with a **6-stage autonomous AI reasoning pipeline** to detect, diagnose, correlate, and remediate incidents across single or multi-cluster environments.

This document describes the high-level system architecture, operational modes, core reasoning engines, and a complete directory breakdown of the repository.

---

## 1. System Overview & Dual-Mode Topology

Claritty operates in two complementary modes to serve both on-call engineers and production cluster fleets:

```text
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                 CLARITTY PLATFORM                               │
├───────────────────────────────────────┬─────────────────────────────────────────┤
│             MODE 1: LOCAL             │           MODE 2: IN-CLUSTER            │
│            (Interactive CLI)          │          (Autonomous Hub-Spoke)         │
├───────────────────────────────────────┼─────────────────────────────────────────┤
│                                       │   Cluster A (Prod)   Cluster B (Dev)    │
│   Developer Workstation / Terminal    │   ┌──────────────┐   ┌──────────────┐   │
│   ┌───────────────────────────────┐   │   │  sre-agent   │   │  sre-agent   │   │
│   │          clarctl-go           │   │   └──────┬───────┘   └──────┬───────┘   │
│   │   (Charmbracelet TUI / CLI)   │   │          │ (Incident JSON)  │           │
│   └──────────────┬────────────────┘   │          └───────┬──────────┘           │
│                  │ Kubeconfig         │                  ▼                      │
│                  ▼                    │      ┌───────────────────────┐          │
│         Kubernetes Cluster API        │      │     sre-agent/hub     │          │
│                  │                    │      │ (Go Server + Svelte)  │          │
│                  ▼                    │      └───────┬───────┬───────┘          │
│         AI Reasoning Engine           │              │       │                  │
│       (Local 6-Stage Pipeline)        │              ▼       ▼                  │
│                  │                    │         PostgreSQL  Slack Alerts        │
│                  ▼                    │              & Web Dashboard            │
│        Interactive TUI Fixes          │                                         │
└───────────────────────────────────────┴─────────────────────────────────────────┘
```

### Mode 1: Local Remediation CLI (`clarctl-go`)
* **Who it is for:** On-call engineers and DevOps practitioners debugging an active issue from their local machine.
* **How it works:** The engineer runs `clarctl scan` or `clarctl watch`. The CLI uses the local `~/.kube/config` to query the active Kubernetes cluster, passes symptoms to the 6-stage AI pipeline, renders root cause analysis directly in the terminal, and prompts interactive step-by-step remediation commands (`y / dry / n`).
* **Storage:** Local SQLite database (`~/.clarctl.db`) for incident history, MTTR metrics, and post-mortems.

### Mode 2: Autonomous In-Cluster Daemon & Hub (`sre-agent`)
* **Who it is for:** Continuous, 24/7 autonomous production monitoring across single or multi-cluster fleets.
* **How it works:** 
  * **Spoke (`sre-agent/agent`):** A lightweight single-replica daemon running inside the cluster. It watches for pod crashes, OOM kills, network partition deadlocks, and API throttling. On failure detection, it runs the 6-stage AI pipeline locally in the cluster and posts structured incident reports to the Hub.
  * **Hub (`sre-agent/hub`):** A centralized aggregator service that persists incidents in PostgreSQL, dispatches Slack notifications, and serves a modern Svelte-based web dashboard.

---

## 2. Repository Directory Breakdown

Below is the complete structural breakdown of every folder in the repository and its role:

```text
claritty/
├── clarctl-go/               # Frontend Go CLI (Production CLI tool)
│   ├── cmd/                  # Cobra CLI commands (scan, watch, status, apply, etc.)
│   ├── internal/             # Core CLI business logic & packages
│   │   ├── ai/               # 6-stage multi-agent pipeline & LLM provider clients
│   │   ├── config/           # CLI configuration loader (~/.clarctl.yaml, env vars)
│   │   ├── db/               # Local SQLite storage & MTTR statistics engine
│   │   ├── incident/         # Incident domain models, timeline & evidence structs
│   │   ├── k8s/              # Kubernetes client-go abstraction wrapper
│   │   ├── runbooks/         # Runbook parsing and execution logic
│   │   └── ui/               # Terminal user interface (Charmbracelet Lipgloss/Huh)
│   ├── runbooks/             # Standard YAML runbooks embedded into binary
│   ├── Makefile              # Build scripts with dynamic version injection
│   └── install.sh            # One-line binary installer script for Linux/macOS
│
├── sre-agent/                # In-Cluster Observability & Incident Platform
│   ├── agent/                # Autonomous in-cluster daemon (Go)
│   │   ├── internal/         # In-cluster AI reasoning, k8s watcher & incident logic
│   │   ├── Dockerfile        # Container build definition for the cluster agent
│   │   └── main.go           # Agent entrypoint with watcher loop & HTTP triggers
│   ├── hub/                  # Centralized aggregator server & web console (Go + Svelte)
│   │   ├── dashboard/        # Frontend web dashboard (Svelte + Vite + Tailwind)
│   │   ├── internal/         # Hub API routes, PostgreSQL persistence, Slack webhook
│   │   ├── Dockerfile        # Multi-stage container build (Svelte + Go server)
│   │   └── main.go           # Hub HTTP server entrypoint (port 8822)
│   ├── deploy/               # Kubernetes deployment manifests
│   │   ├── agent-rbac.yaml   # ServiceAccount, ClusterRole & Binding (Least Privilege)
│   │   ├── agent-configmap.yaml # Cluster name, Hub URL & scan frequency
│   │   └── agent-deployment.yaml # Agent Deployment spec (50m CPU, 64Mi RAM)
│   └── docker-compose.yml    # Local multi-service composition (Hub + Agent + Postgres)
│
├── claritty-agent-client/    # Lightweight Node/Log Telemetry Collector (Go)
│   ├── kube/                 # Kubernetes API & metrics-server client wrappers
│   ├── logs/                 # Container log streaming collector
│   ├── metrics/              # Node-level CPU & memory telemetry collector
│   ├── sender/               # HTTP client transmitting metrics payload to backend
│   └── main.go               # Periodic polling loop (every 10s)
│
├── website/                  # Claritty Documentation & Landing Page
│   ├── docs/                 # Docusaurus documentation markdown pages
│   ├── src/                  # React components, landing pages, and styles
│   └── docusaurus.config.ts  # Docusaurus configuration
│
├── infra/                    # Infrastructure as Code (IaC) & Deployment
│   └── k8s/                  # Helm charts / cloud infrastructure configurations
│
├── tests/                    # Testing Suites & Quality Assurance
│   ├── e2e-api/              # End-to-end API test suites
│   └── load-tests/           # Benchmark and stress testing configurations
│
├── docs/                     # Documentation Assets
│   └── images/               # Logos, CLI GIFs, screenshots, and diagrams
└── (agent-python/ removed)   # Legacy Python prototype has been completely removed
```

---

## 3. Detailed Component Architecture

### 3.1 `clarctl-go/` (The Go CLI)
Built with **Go 1.25.0**, `clarctl-go` provides a lightweight, zero-dependency native binary for engineers.
* **`cmd/`**: Uses `spf13/cobra` to define user commands:
  * `scan`: Runs an on-demand cluster or namespace health diagnosis.
  * `watch`: Launches a live TUI terminal dashboard to monitor ongoing events.
  * `status`: Prints node and workload health summaries.
  * `incidents`: Queries local SQLite database for historical incidents.
  * `show <id>`: Displays full root-cause breakdown, timeline, and evidence for an incident.
  * `apply <id>`: Interactively runs suggested remediation commands with dry-run support.
  * `report <id>`: Exports an incident as structured JSON.
  * `configure`: Interactive setup wizard for LLM providers (Groq, Mistral, OpenAI, Ollama).
* **`internal/ui/`**: Implements terminal graphics using the **Charmbracelet** ecosystem:
  * `lipgloss`: Styling, gradients, boxes, borders, and severity color badges.
  * `huh`: Interactive input forms and confirmation prompts (`y / dry / n`).
  * `bubbletea`: Event-driven terminal dashboard for the `watch` command.
* **`internal/ai/`**: Executes the 6-stage AI pipeline locally against configured LLMs.
* **`internal/k8s/`**: Interacts with the Kubernetes API via `k8s.io/client-go`.

### 3.2 `sre-agent/` (In-Cluster Platform)
Designed for continuous production operation following the Hub-and-Spoke pattern:
* **`agent/` (Spoke):**
  * Deployed inside target Kubernetes clusters as a 1-replica Deployment.
  * Watches cluster events, detects anomalies, and automatically runs the AI diagnosis engine.
  * Pushes completed incident payloads to the Hub via HTTP POST.
  * Exposes HTTP endpoints (`/healthz`, `/trigger`, `/watch/start`, `/watch/stop`) for manual or automated triggers.
* **`hub/` (Hub):**
  * Central aggregation server written in Go.
  * Manages schema migrations and incident persistence in PostgreSQL.
  * Fires webhook notifications to Slack channels with incident summaries and root causes.
  * Serves the compiled Svelte web dashboard directly from embedded static assets on port `8822`.
* **`deploy/`**:
  * Adheres strictly to the **Principle of Least Privilege**: read-only cluster role permissions for inspecting pods, nodes, events, services, and logs.

### 3.3 `claritty-agent-client/` (Telemetry Streamer)
* A specialized Go daemon that periodically queries node metrics and container logs via `k8s.io/client-go` and `k8s.io/metrics`, streaming raw telemetry payloads to a designated metrics endpoint.

---

## 4. The 6-Stage AI Reasoning Pipeline

When an incident is triggered in either `clarctl-go` or `sre-agent/agent`, telemetry flows through a sequential, deterministic multi-agent pipeline:

```text
 ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
 │   Stage 1    │     │   Stage 2    │     │   Stage 3    │
 │ Triage Agent │ ──► │Metrics Agent │ ──► │  Logs Agent  │
 └──────────────┘     └──────────────┘     └──────────────┘
        │                    │                    │
        ▼                    ▼                    ▼
   Scope Issue          CPU / Memory          Log Errors,
  & Failing Pods       Starvation / Spikes   Stack Traces
                                                  │
 ┌──────────────┐     ┌──────────────┐            │
 │   Stage 6    │     │   Stage 5    │     ┌──────▼───────┐
 │  Commander   │ ◄── │Runbook Agent │ ◄── │   Stage 4    │
 │    Agent     │     │              │     │ Infra Agent  │
 └──────────────┘     └──────────────┘     └──────────────┘
        │                    │                    │
        ▼                    ▼                    ▼
  Synthesize RCA,       Match Known         Evaluate PVCs,
 Evidence IDs & Fix     SOP Runbooks        Network, Specs
```

1. **🩺 Triage Agent:** Scans high-level cluster state, detects unhealthy pods (`CrashLoopBackOff`, `OOMKilled`, `Pending`, `ImagePullBackOff`), and scopes the affected namespaces.
2. **📊 Metrics Agent:** Inspects historical and current resource metrics (CPU, Memory, Throttling) for containers and nodes to detect starvation or spikes.
3. **📜 Log Analysis Agent:** Analyzes standard output and error streams, identifying stack traces, panics, connection refused errors, and configuration failures.
4. **🏗 Infrastructure Agent:** Evaluates environment configurations, missing Secrets/ConfigMaps, PVC storage bounds, and Service/DNS reachability.
5. **📖 Runbook Agent:** Checks embedded, battle-tested YAML runbooks (`crash_loop.yaml`, `oom_kill.yaml`, etc.) to map patterns against verified operating procedures.
6. **⚡ Commander Agent:** Synthesizes outputs from all prior stages into a unified `IncidentReport`, attaches concrete Evidence IDs, formulates the Root Cause Analysis (RCA), and generates safe remediation commands.

---

## 5. Reliability & Hallucination Prevention

Claritty employs strict algorithmic guardrails to prevent LLM hallucinations:

### 5.1 Unified Evidence Model
* Every collected signal (log snippet, event message, resource metric, config delta) is assigned an immutable Evidence ID (`[E1]`, `[E2]`, `[E3]`).
* The Commander Agent is strictly instructed to cite evidence IDs when asserting facts in the RCA (e.g., *"Database connection refused [E1] caused by redis-svc endpoint absence [E3]"*).
* Unverified inferences are explicitly marked as **Hypotheses** with manual validation steps.

### 5.2 Incident Timeline Engine
* Automatically constructs a chronological timeline of cluster lifecycle events (`Scheduled`, `Pulled`, `BackOff`, `Failed`), error anomalies, and log alerts leading up to the failure.

### 5.3 "What Changed?" Delta Engine
* Scans cluster mutations within a configurable lookback window (default 1 hour):
  * Workload rollouts and replica scaling
  * Container image tag updates and digest changes
  * ConfigMap and Secret revisions
* Identifies close-proximity changes and tags them as `CORRELATED` causes.

### 5.4 Remediation Safety Guardrails
* All generated remediation commands are checked against a strict allowlist.
* Destructive actions (e.g., deleting namespaces, persistent volumes, or force-killing pods) are flagged with high-visibility warnings and require explicit operator confirmation.

---

## 6. Supported LLM Providers

Both `clarctl-go` and `sre-agent/agent` support pluggable AI inference backends:
* **Groq:** Ultra-fast cloud inference (recommended for sub-second CLI triage).
* **Mistral AI:** High-accuracy reasoning.
* **OpenAI:** GPT-4o / GPT-4o-mini support.
* **Ollama:** 100% air-gapped, local private cluster inference.

---

## 7. Migration from Legacy Python Prototype

Claritty originally started as an experimental Python prototype (`ai-sre` / `agent-python`).

* **Why it was replaced:** Python runtimes introduced heavy distribution overhead (50MB+ PyInstaller packages, slow cold-starts, virtual environment setup friction).
* **Complete Go Rewrite:** The entire platform was rewritten in Go 1.25.0 (`clarctl-go` for the CLI and `sre-agent` for the cluster daemon and hub).
* **Removal:** The legacy `agent-python/` folder has been removed from the repository. All active development, testing, and production deployments are Go-native.

