# 🚀 Claritty AI-SRE — End User Installation Guide

This guide covers setting up the centralized **Hub Server** (Dashboard) and connecting your **Kubernetes Clusters** (Agents).

---

## Part 1: Deploy the Hub Server (Central Dashboard)
You can run the Hub on any Linux server, VM, or your local machine that has Docker installed.

### Step 1: Download the Docker Compose File
On your server, download the official compose file:
```bash
mkdir -p claritty-hub && cd claritty-hub
curl -sL https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/docker-compose.yml -o docker-compose.yml
```

### Step 2: Set Environment Variables & Start
Replace the `DATABASE_URL` with your actual PostgreSQL connection string (from Supabase, Neon, AWS RDS, etc.):
```bash
# 1. Export your Database URL
export DATABASE_URL="postgresql://postgres:yourpassword@your-db-host:5432/claritty?sslmode=require"

# 2. (Optional) Export Slack Webhook for SEV1 alerts
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T000/B000/XXXXXX"

# 3. Start the Hub server in the background
docker-compose up -d
```

### Step 3: Verify the Hub is Live
Open your web browser and navigate to:
👉 `http://<your-server-ip>:8822`

*(You will see the Claritty Dashboard live, waiting for cluster agents to connect).*

---

## Part 2: Connect Your Kubernetes Cluster (The Agent)
Now, switch to your terminal where your `kubectl` is configured for your Kubernetes cluster (e.g., EKS, GKE, AKS, or Minikube).

### Step 1: Download the Agent Manifests
Download the 3 required Kubernetes deployment files:
```bash
mkdir -p claritty-agent && cd claritty-agent

curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-rbac.yaml
curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-configmap.yaml
curl -O https://raw.githubusercontent.com/Vaishnav88sk/claritty/master/sre-agent/deploy/agent-deployment.yaml
```

### Step 2: Configure Your Cluster Name & Hub URL
Edit the `agent-configmap.yaml` file. Set your `CLARITTY_CLUSTER_NAME` and point `CLARITTY_HUB_URL` to the IP/domain of the Hub server you started in Part 1.

```bash
# Open the file in your editor (nano, vim, etc.)
nano agent-configmap.yaml
```

Make sure the `data` section looks like this:
```yaml
data:
  CLARITTY_CLUSTER_NAME: "production-us-east"       # UNIQUE name for this K8s cluster
  CLARITTY_HUB_URL: "http://<your-hub-ip>:8822"     # IP or Domain of your Hub Server
  LLM_PROVIDER: "groq"
  LLM_MODEL: "llama-3.3-70b-versatile"
  SCAN_INTERVAL_SECS: "300"
```

### Step 3: Apply RBAC & ConfigMap
Run the following commands to create the namespace, permissions, and configuration:
```bash
kubectl apply -f agent-rbac.yaml
kubectl apply -f agent-configmap.yaml
```

### Step 4: Create the AI API Key Secret
Create a secure Kubernetes Secret containing your AI provider API key (Groq, OpenAI, or Mistral):
```bash
# Replace 'gsk_your_api_key_here...' with your actual API key
kubectl create secret generic claritty-agent-secrets \
  -n claritty \
  --from-literal=GROQ_API_KEY=gsk_your_api_key_here...
```

### Step 5: Deploy the Agent Pod
Deploy the lightweight agent into your cluster:
```bash
kubectl apply -f agent-deployment.yaml
```

---

## Part 3: Verify the Connection
Check the logs of the agent pod to verify it successfully connected to your K8s API and the Hub:
```bash
kubectl logs -n claritty deployment/claritty-agent -f
```

**Expected Log Output:**
```text
Claritty SRE Agent starting | cluster=production-us-east hub=http://<your-hub-ip>:8822
[production-us-east] Starting AI scan...
[production-us-east] Scan complete: SEV4 - Cluster is healthy (confidence 95%)
Report sent to hub: 201 Created
```

🎉 **That's it!** Go back to your browser (`http://<your-hub-ip>:8822`). You will instantly see `production-us-east` on your dashboard!
