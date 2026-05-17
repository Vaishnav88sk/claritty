# Clarctl CLI Local AI-SRE for Kubernetes

`clarctl` is the standalone, command-line interface for Claritty. It runs directly on your local machine, connects to your active Kubernetes cluster via your `kubeconfig`, and uses a 6-stage AI agent collaborative pipeline to instantly diagnose root causes and offer interactive auto-remediation.

---

## 🚀 Installation (For End Users)

You do not need to install Go or build from source to use `clarctl`. You can download the pre-compiled standalone binary directly:

### Linux / macOS (curl)

```bash
# 1. Download the latest binary
curl -sL https://github.com/Vaishnav88sk/claritty/releases/latest/download/clarctl-linux-amd64 -o clarctl

# 2. Make it executable
chmod +x clarctl

# 3. Move it to your PATH
sudo mv clarctl /usr/local/bin/

# 4. Verify installation
clarctl -h
```

---

## ⚙️ Configuration

`clarctl` requires an AI provider API key to perform root cause analysis. It supports Groq, OpenAI, and Mistral.

```bash
# Set your preferred provider (default is groq)
export LLM_PROVIDER="groq"
export LLM_MODEL="llama-3.3-70b-versatile"

# Provide your API Key
export GROQ_API_KEY="gsk_your_api_key_here..."
```

*(If using OpenAI, set `LLM_PROVIDER="openai"` and `OPENAI_API_KEY="sk-..."`).*

---

## 💻 Usage & Commands

`clarctl` automatically uses your current `kubectl` context (e.g. `~/.kube/config`).

### 1. Scan an Entire Namespace
Scans all pods, deployments, and events in a namespace for issues:
```bash
clarctl scan namespace production
```

### 2. Scan a Specific Pod
Focuses the 6-stage AI analysis on a single problematic pod:
```bash
clarctl scan pod payment-service-5b689 -n production
```

### 3. Interactive Auto-Remediation
When `clarctl` finds a Root Cause, it generates a step-by-step remediation plan. For each step, it will prompt you:
```text
Execute? [y/N/dry]:
```
- `y`: Executes the kubectl command directly against your cluster.
- `dry`: Runs the command with `--dry-run=client` to validate it safely.
- `N`: Skips the command.

---

## 🛠️ Getting Started (For Developers)

If you want to contribute, modify the AI prompts, or build the CLI from source:

```bash
# 1. Clone the repository
git clone https://github.com/Vaishnav88sk/claritty.git
cd claritty/clarctl-go

# 2. Install dependencies
go mod tidy

# 3. Build the binary locally
go build -o clarctl .

# 4. Run it locally
./clarctl scan namespace default
```

---

## 🏗️ Architecture

```text
┌────────────────────────────────────────────────────────┐
│                     Local Machine                      │
│                                                        │
│   ┌───────────┐      ┌─────────────┐     ┌─────────┐   │
│   │ SRE Eng.  │ ───► │ clarctl CLI │ ──► │ AI Pipe │   │
│   └───────────┘      └──────┬──────┘     └────┬────┘   │
└─────────────────────────────┼─────────────────┼────────┘
                              │                 │         
                              ▼                 ▼         
                       ┌─────────────┐   ┌─────────────┐  
                       │ K8s Cluster │   │ LLM Provider│  
                       └─────────────┘   └─────────────┘  
```

1. **Triage Agent**: Determines the scope of the scan.
2. **Metrics Agent**: Collects pod/node CPU and memory utilization.
3. **Log Agent**: Fetches tail logs for failing containers.
4. **Infra Agent**: Inspects K8s events and exit codes.
5. **Runbook Agent**: Matches failure patterns against embedded YAML runbooks.
6. **Incident Commander**: Synthesizes all telemetry into a final Root Cause summary and remediation plan.
