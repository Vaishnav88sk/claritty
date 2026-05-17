# Claritty - AI-SRE for Kubernetes
### Production-grade AIOps platform for cluster observability, incident response & auto-remediation

Claritty is an **open-source, cloud-native AI Site Reliability Engineering platform** for Kubernetes clusters.
It combines real-time cluster telemetry with a **6-stage AI agent pipeline** to automatically detect, diagnose, and remediate incidents, reducing MTTR from hours to minutes.

---

## 🌟 The Two Modes of Claritty

Claritty provides two powerful ways to interact with your Kubernetes infrastructure, depending on your needs:

### 1. Clarctl CLI (Local Remediation Tool)
A powerful command-line interface run from your local machine. It connects to your current Kubernetes context to instantly analyze namespaces or specific pods, generate an RCA (Root Cause Analysis), and offer interactive, step-by-step remediation commands. Perfect for on-call engineers debugging live incidents.

### 2. SRE Agent & Hub (Centralized Platform)
A lightweight, in-cluster daemon (the Agent) that continuously monitors your infrastructure. It autonomously performs the 6-stage AI pipeline on failing resources and pushes structured incident reports to a centralized Hub server. The Hub provides a beautiful web dashboard for a multi-cluster overview, Slack alerts, and detailed RCA records. Perfect for production monitoring.

---

## ✨ Features

- 📊 **Node-level & Pod-level Metrics**: Real-time CPU, memory, and resource usage collection.
- ⚡ **Auto Incident Detection**: Detects complex cascading failures, API server throttling, DNS resolution timeouts, Split-Brain StatefulSets, network partition deadlocks, alongside standard CrashLoopBackOff, OOMKilled, and Pending states.
- 🧠 **6-Stage AI Agent Pipeline**: Triage -> Metrics -> Logs -> Infra -> Runbook -> Commander agents collaboratively diagnose root causes.
- 🚨 **Interactive Auto-Remediation (CLI)**: Proposes step-by-step kubectl fixes locally. Prompts `y / dry / n` before executing anything.
- 🌐 **Centralized Dashboard (Agent)**: Web UI to view multi-cluster health, active incidents, and automated remediation plans.
- 🔒 **Safety First**: All remediation commands are validated against a strict allowlist. Destructive commands are flagged.
- 📖 **Built-in Runbooks**: Battle-tested YAML runbooks for common failure modes embedded directly in the logic.
- 🗄 **Incident History**: Database-backed incident logging with MTTR tracking and status lifecycle.

---

## 🚀 Installation (For End Users)

You can quickly install and deploy Claritty without needing to build from source. Choose the mode you want to use.

### Option 1: Install Clarctl CLI (Local Tool)
Download the pre-compiled binary to your local machine to diagnose your clusters instantly:

```bash
# 1. Download the latest binary (Linux/macOS)
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/clarctl-go/clarctl-go/install.sh | bash
```
```bash
# 2. Run help

clarctl -h
```
```bash
# 3. Run a scan!
clarctl scan
```

### Option 2: Deploy the SRE Agent & Hub (In-Cluster)
Deploy the centralized dashboard and the agent into your clusters for continuous monitoring. For detailed steps, see [INSTALLATION.md](INSTALLATION.md).

**Start the Hub Server (Dashboard)**
```bash
# Run the Hub via Docker Compose
export DATABASE_URL="postgresql://user:pass@host:5432/claritty?sslmode=require"
```
```bash
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/docker-compose.yml -o docker-compose.yml
docker-compose up -d
# View dashboard at http://localhost:8822
```

**Deploy the Agent to your Clusters**
```bash
# Apply agent manifests
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-configmap.yaml
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-deployment.yaml
```
*(Remember to update the ConfigMap with your specific Hub IP and Cluster Name!)*

---

## 🏗️ Architecture

### Clarctl CLI Architecture
Runs locally on the engineer's machine.
`Developer Terminal -> clarctl -> Kubeconfig -> K8s API -> AI Pipeline -> Terminal Output`

### SRE Agent & Hub Architecture (Hub-Spoke Model)
```text
Cluster A (prod) ──► claritty-agent ─┐
Cluster B (dev)  ──► claritty-agent ─┼──► Hub Server (port 8822) ──► Web Dashboard + Slack Alerts
Cluster C (qa)   ──► claritty-agent ─┘         │
                                    PostgreSQL Database
```

---

## 🛠️ Getting Started (For Developers)

If you want to contribute, modify the code, or build from source:

```bash
# Clone the repository
git clone https://github.com/Vaishnav88sk/claritty.git
cd claritty

# Building the CLI
cd clarctl-go
go mod tidy
go build -o clarctl .

# Running the Hub from source
cd ../sre-agent/hub
export DATABASE_URL="postgresql://user:pass@host:5432/claritty"
go run .

# Running the Agent from source locally
cd ../agent
export CLARITTY_CLUSTER_NAME="local-dev"
export CLARITTY_HUB_URL="http://localhost:8822"
export GROQ_API_KEY="your_key_here"
go run .
```

---

## 💻 Sample Examples & Output

### CLI Interactive RCA
Running `clarctl scan namespace prod` when a pod is crash-looping:

```text
[Claritty] Scanning namespace 'prod'...
[!] Detected issue: payment-service-84f9b8c-x2z9 (CrashLoopBackOff)
[AI Pipeline] Triage -> Logs -> Metrics -> Infra -> Commander...

🚨 ROOT CAUSE (SEV 1 - 95% Confidence):
The payment-service pod is failing to start because it cannot connect to the Redis cache at 'redis.prod.svc.cluster.local:6379'. Connection refused.

🔧 PROPOSED REMEDIATION:
Step 1: Check if the Redis service is running.
Command: kubectl get svc redis -n prod
Execute? [y/N/dry]: y
...
```

![Claritty CLI Output Demo](docs/images/claritty-clarctl-1.png)

### Hub Dashboard Incident Card
When the `sre-agent` runs in the cluster, it pushes structured JSON to the Hub:
```json
{
  "cluster": "prod-us-east",
  "namespace": "billing",
  "severity": "SEV2",
  "title": "OOMKilled Event on Invoice Generator",
  "root_cause": "Container 'worker' exceeded its memory limit of 512Mi. Last usage spike reached 512.4Mi during a large PDF generation task.",
  "remediation_plan": [
    {
      "step_number": 1,
      "description": "Increase memory limits for the invoice deployment.",
      "command": "kubectl set resources deployment invoice-generator -n billing --limits=memory=1Gi",
      "is_destructive": false
    }
  ]
}
```
![Claritty Hub Dashboard Output Demo](docs/images/claritty-ai-sre-1.png)

---

## 📋 Incident Categories Detected

Claritty's pipeline is trained to handle a vast array of Kubernetes failure states:

- **Pod Lifecycle Failures**: `CrashLoopBackOff`, `ImagePullBackOff`, `CreateContainerConfigError`.
- **Resource Starvation**: `OOMKilled`, CPU Throttling, Node Disk Pressure.
- **Network Issues**: Service resolution failures, DNS timeouts, missing endpoints.
- **Storage Issues**: Unbound PersistentVolumeClaims, mounting failures.
- **RBAC & Security**: Unauthorized API calls, missing service account permissions.

---

## ⚖️ Comparison with Industry Tools

| Feature | Claritty | Datadog / New Relic | Prometheus/Thanos | Robusta |
|---|---|---|---|---|
| In-cluster agent | ✅ Deployment 1 replica | ✅ | ✅ | ✅ |
| AI-powered RCA | ✅ 6-stage LLM pipeline | ❌ (Mostly manual) | ❌ | Partial |
| Multi-cluster hub | ✅ Open Source Hub | ✅ SaaS | ✅ Thanos | ✅ SaaS |
| Self-hosted | ✅ | ❌ SaaS only | ✅ | Partial |
| Cost | Free / Open Source | $$$$ | Free | Free/Paid |

---

## 📍 Checkpoints & Future Roadmap

- [x] CLI for local cluster diagnosis.
- [x] Multi-agent collaborative LLM pipeline.
- [x] Agent deployment for continuous in-cluster monitoring.
- [x] Hub server & Web UI for multi-cluster overview.
- [x] PostgreSQL persistence & Slack integration.
- [ ] **Add complete K8s observability next** (Custom metrics, distributed tracing integration, eBPF network flows).

---

*Claritty is actively maintained and built for modern SRE teams. Contributions and feedback are welcome!*