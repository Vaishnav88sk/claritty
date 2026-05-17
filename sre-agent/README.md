# Claritty SRE Agent & Hub - In-Cluster Observability

This directory contains the production-grade, in-cluster continuous monitoring solution for Claritty. It follows a **Hub-Spoke architecture** where lightweight agents run inside your Kubernetes clusters and push telemetry/RCA reports to a centralized Hub server with a beautiful web dashboard.

---

## 🚀 Installation (For End Users)

You can easily deploy the Hub server and cluster agents using pre-built Docker images. No Go development environment required.

### Part 1: Start the Hub Server (Central Dashboard)
Run the centralized Hub server on any Linux machine or VM with Docker Compose:

```bash
mkdir -p claritty-hub && cd claritty-hub

# 1. Download the official docker-compose.yml
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/docker-compose.yml -o docker-compose.yml

# 2. Set your PostgreSQL database connection string
export DATABASE_URL="postgresql://postgres:password@your-db-host:5432/claritty?sslmode=require"

# 3. (Optional) Set Slack webhook for SEV1/SEV2 alerts
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T000/B000/XXXXXX"

# 4. Start the Hub
docker-compose up -d
```
👉 Open **http://localhost:8822** in your browser. The dashboard is live!

---

### Part 2: Deploy the Agent to Your Kubernetes Clusters
Run the following commands in your terminal configured with `kubectl` for your cluster:

```bash
mkdir -p claritty-agent && cd claritty-agent

# 1. Download the Agent manifests
curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-rbac.yaml
curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-configmap.yaml
curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-deployment.yaml

# 2. Edit agent-configmap.yaml
#    Set CLARITTY_CLUSTER_NAME (e.g. "prod-us-east")
#    Set CLARITTY_HUB_URL (e.g. "http://<hub-server-ip>:8822")
nano agent-configmap.yaml

# 3. Apply RBAC and ConfigMap
kubectl apply -f agent-rbac.yaml
kubectl apply -f agent-configmap.yaml

# 4. Create AI API Key Secret (Replace with your Groq/OpenAI/Mistral key)
kubectl create secret generic claritty-agent-secrets \
  -n claritty \
  --from-literal=GROQ_API_KEY=gsk_your_api_key_here...

# 5. Deploy the Agent
kubectl apply -f agent-deployment.yaml
```
*(Repeat Part 2 for every Kubernetes cluster you want to monitor. They will all appear on your Hub dashboard automatically).*

---

## 🏗️ Architecture (Hub-Spoke Model)

```text
Cluster A (prod) ──► claritty-agent ─┐
Cluster B (dev)  ──► claritty-agent ─┼──► Hub Server (port 8822) ──► Web Dashboard + Slack Alerts
Cluster C (qa)   ──► claritty-agent ─┘         │
                                    PostgreSQL Database
```

- **Agent (`sre-agent/agent`)**: Runs as a 1-replica K8s Deployment. Uses `InClusterConfig` to monitor pod/node health. Consumes ~50m CPU and ~64Mi RAM.
- **Hub (`sre-agent/hub`)**: Central Go REST API server. Receives incident payloads from agents, persists them in PostgreSQL, triggers Slack alerts, and serves the static SPA dashboard.
- **Dashboard (`sre-agent/hub/dashboard`)**: Vanilla JS Single Page Application. Features multi-cluster health cards, namespace filtering, MTTR analytics, and an interactive RCA modal with copy-able remediation commands.

---

## 🛠️ Getting Started (For Developers)

If you want to build the Hub or Agent from source or modify the code:

### 1. Running the Hub Locally (Source)
```bash
cd sre-agent/hub
go mod tidy
export DATABASE_URL="postgresql://user:pass@localhost:5432/claritty"
go run .
# API runs on :8822/api/v1/
# Dashboard runs on :8822/
```

### 2. Running the Agent Locally (Source)
The agent will automatically use your local `~/.kube/config` when run outside a cluster:
```bash
cd sre-agent/agent
go mod tidy
export CLARITTY_CLUSTER_NAME="local-dev"
export CLARITTY_HUB_URL="http://localhost:8822"
export GROQ_API_KEY="your_groq_api_key"
go run .
```

---

## 📡 Agent HTTP Endpoints (Port 9090)

The in-cluster agent exposes a lightweight HTTP server for K8s probes and Hub control:

| Endpoint | Method | Description |
|---|---|---|
| `/healthz` | GET | Liveness/Readiness probe |
| `/trigger` | POST | Triggers one immediate AI scan of the cluster |
| `/watch` | POST | Starts continuous background scanning (default: 5 min) |
| `/watch` | DELETE | Stops continuous background scanning |
| `/status` | GET | Returns agent configuration and watch state |

---

## 🌐 Hub API Endpoints (Port 8822)

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/clusters` | GET | List all connected clusters and health scores |
| `/api/v1/clusters/:name` | GET | Get single cluster details |
| `/api/v1/incidents` | GET | List incidents (Supports `?cluster=`, `?severity=`, `?status=`) |
| `/api/v1/incidents` | POST | Webhook for agents to push new incident reports |
| `/api/v1/incidents/:id` | GET | Get full incident RCA and remediation plan |
| `/api/v1/incidents/:id/status` | PATCH | Update incident status (`MITIGATED` or `RESOLVED`) |
| `/api/v1/stats` | GET | Get global MTTR and incident counts for analytics |
