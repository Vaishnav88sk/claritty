# Claritty SRE Agent

Production-grade, AI-powered Site Reliability Engineering system for Kubernetes. Runs **inside your clusters** as a lightweight agent, performs 6-stage AI root-cause analysis, and sends structured incident reports to a centralized hub with a beautiful web dashboard.

---

## Architecture

```
Cluster A ──► claritty-agent ─┐
Cluster B ──► claritty-agent ─┼──► Hub Server (port 8822) ──► Dashboard + Slack
Cluster C ──► claritty-agent ─┘         │
                                    PostgreSQL
```

- **Agent**: 1 replica per cluster (Deployment). Reads cluster state via `InClusterConfig`. ~50m CPU, ~64Mi RAM.
- **Hub**: External Go server. Receives incidents, serves dashboard on `:8822`.
- **Dashboard**: Single-page app. Multi-cluster view, namespace drill-down, RCA detail, remediation steps.
- **Storage**: PostgreSQL (Supabase, Neon, Railway, or self-hosted).

---

## Quick Start

### 1. Set up PostgreSQL

Get a free database at [Supabase](https://supabase.com) or [Neon](https://neon.tech). Copy your connection string.

### 2. Run the Hub (Local)

```bash
export DATABASE_URL="postgresql://user:pass@host:5432/claritty"
export SLACK_WEBHOOK_URL="https://hooks.slack.com/..."   # optional
docker-compose up
```

Open **http://localhost:8822** — dashboard is live.

### 3. Deploy the Agent (Per Cluster)

```bash
# 1. Edit deploy/agent-configmap.yaml
#    Set CLARITTY_CLUSTER_NAME and CLARITTY_HUB_URL

# 2. Apply RBAC and config
kubectl apply -f deploy/agent-rbac.yaml
kubectl apply -f deploy/agent-configmap.yaml

# Set your GROQ_API_KEY in the secret
kubectl create secret generic claritty-agent-secrets \
  -n claritty \
  --from-literal=GROQ_API_KEY=your_key_here

# 3. Deploy the agent
kubectl apply -f deploy/agent-deployment.yaml

# 4. Verify it's running
kubectl logs -n claritty deployment/claritty-agent -f
```

Repeat for every cluster — each one will appear in the dashboard automatically.

---

## Triggering Scans (from Dashboard)

Each cluster card in the dashboard has two buttons:

| Button | What it does |
|---|---|
| **🔍 Scan Once** | Triggers one immediate AI scan of the cluster |
| **▶ Watch** | Starts continuous scanning (every 5 min by default, configurable) |
| **⏸ Stop Watching** | Stops the continuous scan loop |

---

## Agent HTTP Endpoints

The agent exposes a lightweight HTTP server (default port `9090`) for control:

| Endpoint | Method | Description |
|---|---|---|
| `/healthz` | GET | Liveness check (used by K8s probes) |
| `/trigger` | POST | Start one immediate scan |
| `/watch` | POST | Start continuous scanning |
| `/watch` | DELETE | Stop continuous scanning |
| `/status` | GET | Returns agent state (cluster name, watch mode, interval) |

---

## Hub API Endpoints

All served on port `8822`:

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/clusters` | GET | List all clusters + health |
| `/api/v1/clusters/:name` | GET | Single cluster detail |
| `/api/v1/clusters/:name/incidents` | GET | Incidents for one cluster |
| `/api/v1/incidents` | GET | All incidents (filter by cluster, ns, severity, status) |
| `/api/v1/incidents` | POST | Receive incident from agent |
| `/api/v1/incidents/:id` | GET | Full incident + RCA |
| `/api/v1/incidents/:id/status` | PATCH | Update status (MITIGATED/RESOLVED) |
| `/api/v1/stats` | GET | Global MTTR, counts |

---

## Configuration

### Agent (environment variables / ConfigMap)

| Variable | Default | Description |
|---|---|---|
| `CLARITTY_CLUSTER_NAME` | `default-cluster` | Unique name for this cluster |
| `CLARITTY_HUB_URL` | `http://localhost:8822` | External hub URL |
| `LLM_PROVIDER` | `groq` | `groq` / `openai` / `mistral` |
| `LLM_MODEL` | `llama-3.3-70b-versatile` | Model name |
| `GROQ_API_KEY` | — | API key (mount from Secret) |
| `SCAN_INTERVAL_SECS` | `300` | Seconds between continuous scans |
| `WATCH_NAMESPACES` | *(all)* | Comma-separated namespaces to monitor |

### Hub (environment variables)

| Variable | Default | Description |
|---|---|---|
| `HUB_PORT` | `8822` | Dashboard + API port |
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `SLACK_WEBHOOK_URL` | — | Slack incoming webhook (optional) |
| `SLACK_CHANNEL` | `#sre-alerts` | Slack channel for SEV1/SEV2 alerts |
| `CLARITTY_HUB_API_KEY` | — | Shared secret for agent→hub auth (optional) |

---

## Local Development (without Docker)

```bash
# Hub
cd hub
go mod tidy
DATABASE_URL="postgresql://..." go run .

# Agent (connects to your local cluster via ~/.kube/config)
cd agent
go mod tidy
CLARITTY_CLUSTER_NAME=local \
CLARITTY_HUB_URL=http://localhost:8822 \
GROQ_API_KEY=your_key \
go run .
```

---

## How It Compares to Industry Tools

| Feature | Claritty SRE | Datadog | Prometheus/Thanos | Robusta |
|---|---|---|---|---|
| In-cluster agent | ✅ Deployment 1 replica | ✅ | ✅ | ✅ |
| AI-powered RCA | ✅ 6-stage LLM pipeline | ❌ | ❌ | Partial |
| Multi-cluster hub | ✅ | ✅ SaaS | ✅ Thanos | ✅ SaaS |
| Self-hosted | ✅ | ❌ SaaS only | ✅ | Partial |
| Open source | ✅ | ❌ | ✅ | ✅ |
| Cost | Free | $$$$ | Free | Free/Paid |
