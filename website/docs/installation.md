---
sidebar_position: 2
---

# Installation

Claritty is designed to be deployed effortlessly. Depending on your operational requirements, you can install the **Clarctl CLI** for local, on-demand diagnostics, or deploy the **SRE Agent & Hub** for continuous, in-cluster observability.

---

## Option 1: Install Clarctl CLI (Local Tool)

The `clarctl` CLI is a standalone binary that connects to your local `kubeconfig` and interacts directly with your clusters from your terminal.

### 1. Download the Binary
You can install the latest release directly via our installation script (supports Linux and macOS):

```bash
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/clarctl-go/install.sh | bash
```

### 2. Verify Installation
Ensure the binary is in your path and correctly installed:

```bash
clarctl version
```

### 3. Run a Scan
Instantly diagnose your current Kubernetes context:

```bash
clarctl scan
```

---

## Option 2: Deploy the SRE Agent & Hub (In-Cluster)

For continuous, 24/7 monitoring, deploy the Agent into your clusters and spin up the Centralized Hub.

### 1. Start the Hub Server
The Hub serves as the central control plane, receiving telemetry from your agents and hosting the web dashboard. You can spin this up quickly using Docker Compose.

```bash
# Export your database credentials
export DATABASE_URL="postgresql://user:pass@host:5432/claritty?sslmode=require"

# Download the compose file
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/docker-compose.yml -o docker-compose.yml

# Start the Hub in detached mode
docker-compose up -d
```
> View the beautiful Claritty dashboard at `http://localhost:8822`!

### 2. Deploy the SRE Agent
Next, deploy the lightweight SRE agent directly into the Kubernetes clusters you wish to monitor.

```bash
# 1. Apply the required RBAC permissions
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-rbac.yaml

# 2. Apply the configuration (Make sure to edit this file with your Hub IP!)
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-configmap.yaml

# 3. Deploy the agent daemon
kubectl apply -f https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-deployment.yaml
```

:::warning

Before applying `agent-configmap.yaml`, ensure you replace the default placeholders with your specific Hub Server IP and a unique `Cluster Name` to identify it on the dashboard.

:::
