# Claritty — AI-SRE for Kubernetes
### Production-grade AIOps platform for cluster observability, incident response & auto-remediation

Claritty is an **open-source, cloud-native AI Site Reliability Engineering platform** for Kubernetes clusters.
It combines real-time cluster telemetry with a **6-stage AI agent pipeline** to automatically detect, diagnose, and remediate incidents — reducing MTTR from hours to minutes.

---

## ✨ Features

- 📊 **Node-level metrics**: Real-time CPU, memory, and resource usage via [`client-go`](https://github.com/kubernetes/client-go) and the Kubernetes metrics-server.
- ⚡ **Auto incident detection**: Detects CrashLoopBackOff, OOMKilled, Pending pods, node pressure, image pull errors and more.
- 🧠 **6-Stage AI Agent Pipeline**: Triage → Metrics → Logs → Infra → Runbook → Commander agents collaboratively diagnose root causes.
- 🚨 **Interactive Auto-Remediation**: Proposes step-by-step kubectl fixes. Prompts `y / dry / n` before executing anything.
- 🔒 **Safety First**: All remediation commands validated against a strict allowlist before execution.
- 📖 **Built-in Runbooks**: 7 battle-tested YAML runbooks for common failure modes (crashloop, OOM, disk pressure, etc.) embedded directly in the binary.
- 🗄 **Incident History**: SQLite-backed incident database with full JSON export, MTTR tracking, and status lifecycle.
- 🔧 **Extensible**: Supports Groq, OpenAI, and Mistral LLMs. Configurable thresholds, namespaces, and scan intervals.

---

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      clarctl CLI                        │
│              (Single ~40MB Go binary)                   │
└──────────────────────────┬──────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
   ┌──────────────┐ ┌────────────┐ ┌──────────────┐
   │ Kubernetes   │ │ AI Agent   │ │   SQLite DB  │
   │ client-go    │ │ Pipeline   │ │  ~/.claritty │
   │ (pods/events │ │ (6 stages) │ │  /clarctl.db │
   │  /nodes/logs)│ │ langchaingo│ └──────────────┘
   └──────────────┘ └────────────┘
                           │
              ┌────────────▼────────────┐
              │   LLM API (Groq/OpenAI/ │
              │   Mistral) — Cloud only  │
              └─────────────────────────┘

Agent Pipeline:
  Triage → Metrics → Logs → Infra → Runbook → Commander
```

---

## 🚀 Install clarctl (One Line)

> **No Python. No cloning. No virtual environments.**
> Just download and run.

```bash
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/clarctl-go/clarctl-go/install.sh | bash
```

This automatically:
1. Detects your OS and CPU architecture (Linux/Mac, amd64/arm64)
2. Downloads the correct pre-compiled binary from GitHub Releases
3. Places `clarctl` in `~/.local/bin/clarctl`
4. Creates a `~/.claritty/.env` config template

**Binary sizes:** `~40MB` — no runtime dependencies required.

### Configure your API key

Add your LLM API key to `~/.claritty/.env`:

```bash
# Get a free key at https://console.groq.com
GROQ_API_KEY=your_key_here
```

---

## 🖥 Usage

```bash
clarctl status              # Show live cluster health dashboard
clarctl scan                # Run a single AI-SRE scan
clarctl scan --apply        # Scan + interactively apply remediation
clarctl watch               # Continuous monitoring loop (Ctrl+C to stop)
clarctl watch --apply       # Watch + auto-prompt remediation on SEV1/2
clarctl incidents           # View incident history
clarctl incidents --severity SEV1 --hours 48
clarctl show INC-ABCD1234   # Show detailed incident report
clarctl apply INC-ABCD1234  # Apply remediation for a saved incident
clarctl report INC-ABCD1234 -o report.json  # Export as JSON
```

---

## 🏗 Getting Started (Full Stack)

### Prerequisites

- Kubernetes cluster (k3d, minikube, EKS, GKE, or any CNCF-compliant distro)
- `kubectl` configured and working
- A free [Groq API key](https://console.groq.com) (or OpenAI/Mistral)

### 1. Deploy the Claritty Monitoring Agent

```bash
kubectl apply -f deploy/kubernetes/agent-daemonset.yaml
```

### 2. Backend Setup (Optional)

Run the Go backend service for metrics dashboard:

```bash
cd backend/
go run main.go
```

### 3. Install clarctl AI-SRE Engine

```bash
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/clarctl-go/clarctl-go/install.sh | bash
```

### 4. Run your first scan

```bash
clarctl scan --apply
```

---

## 📦 Build from Source

```bash
git clone https://github.com/Vaishnav88sk/claritty.git
cd claritty/clarctl-go

# Build for current platform
make build

# Cross-compile for all platforms (Linux, Mac, Windows)
make cross

# Install to /usr/local/bin
sudo make install
```

---

## 📈 Incident Categories Detected

| Category | Description |
|:---|:---|
| `crashloop` | CrashLoopBackOff pods — exit code analysis & restart |
| `oom` | OOMKilled containers — memory limit diagnosis |
| `high_cpu` | CPU pressure — throttling & spike detection |
| `image_pull` | ImagePullBackOff — registry auth & image existence checks |
| `pending` | Stuck Pending pods — scheduling constraint diagnosis |
| `node_not_ready` | Node health failures — taint, pressure, kubelet issues |
| `disk_pressure` | Disk pressure on nodes — cleanup recommendations |

---

## 🔮 Roadmap

- [x] AI-driven anomaly detection on metrics
- [x] Root-cause analysis (RCA) with alert summary generation
- [x] Auto incident response & remediation
- [x] **`clarctl` — standalone Go CLI binary (~40MB)**
- [x] 6-stage AI agent pipeline (Triage/Metrics/Logs/Infra/Runbook/Commander)
- [x] SQLite-backed incident history & MTTR tracking
- [ ] Namespace-level observability
- [ ] Multi-cluster support
- [ ] Prometheus & Loki deep integration
- [ ] Web dashboard with filtering & sorting
- [ ] Integration with OpenTelemetry (OTel)
- [ ] Slack / PagerDuty alerting

---

## 🤝 Contributing

Contributions are welcome! Please open an issue or submit a PR.

- **Python AI-SRE engine:** `ai-sre/` directory, `vaishnav-claritty` branch
- **Go CLI rewrite:** `clarctl-go/` directory, `clarctl-go` branch

---

## 📜 License

[MIT LICENSE](/LICENSE)