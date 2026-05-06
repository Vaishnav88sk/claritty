# Claritty - Observability and Monitoring
### AIOps Agent for Kubernetes clusters, CI/CD, and Anomaly Detection

Claritty is an **open-source, lightweight, cloud-native observability agent** for Kubernetes clusters.
It collects **real-time node and custom metrics**, forwards them to a backend service for visualization and analysis, and applies **AI-driven root-cause analysis (RCA)** to detect spikes, failures, and trigger intelligent alert summaries.
Claritty is built to be **scalable, extensible, and easy to deploy** across any Kubernetes environment - evolving into a full **AIOps-driven observability platform** for end-to-end DevOps monitoring.

---

## ✨ Features

- 📊 **Node-level metrics**: Collects CPU, memory, and other resource usage using [`client-go`](https://github.com/kubernetes/client-go) and the Kubernetes **metrics-server**.
- ⚡ **Custom metrics**: Fetches additional workload and API-driven metrics using the Kubernetes API.
- 🛰 **DaemonSet deployment**: Runs on every node to ensure cluster-wide coverage.
- 🌐 **Backend integration**: Sends metrics to a backend service (Go + Gin) hosted on EC2 for processing and dashboard visualization.
- 🧠 **AI-based RCA**: Python-powered root-cause analysis engine detects metric spikes and failures, correlates signals, and generates human-readable alert summaries.
- 🚨 **Auto incident response**: Automatically triggers alerts and incident summaries when anomalies are detected - reducing mean time to resolution (MTTR). Supports ML-model based decision making to take automated corrective actions when permitted.
- 🔧 **Extensible design**: Can be extended for namespace-level, multi-node, and multi-cluster observability.

---

## 🏗 Architecture 
```
+------------------+       +-----------------+       +----------------------+
| Claritty Agent   | --->  | Backend (Go/Gin)| --->  | Dashboard (Frontend) |
| (DaemonSet Pods) |       |  on EC2         |       |  Metrics + Charts    |
+------------------+       +-----------------+       +----------------------+

Data sources: client-go + metrics-server + K8s custom metrics API
```

---

## 🚀 Getting Started

### Prerequisites

- Kubernetes cluster (k8s, k3d, minikube, EKS, GKE, or any CNCF-compliant distro)
- `kubectl` access
- [metrics-server](https://github.com/kubernetes-sigs/metrics-server) installed
- Python 3.11+ (for the RCA engine)

### 1. Deploy Claritty Agent

```bash
kubectl apply -f agent-daemonset.yaml
```

### 2. Backend Setup

Run the Go backend service (on EC2 or locally):

```bash
go run main.go
```

### 3. Start the RCA Engine

```bash
cd ai-sre/
python sre_swarm_new.py
```

### 4. View Metrics & Alerts

Access the dashboard to visualize node metrics, anomaly events, and RCA-generated alert summaries.

---

## 📈 Example Metrics Collected

- Node name
- CPU usage (%)
- Memory usage (MB)
- Node metrics via Kubernetes API
- Anomaly events with RCA summaries
- And many more custom metrics you want 😉

---

## 🔮 Roadmap

- [x] AI-driven anomaly detection on metrics
- [x] Root-cause analysis (RCA) with alert summary generation
- [x] Auto incident response & remediation
- [ ] Namespace-level observability
- [ ] Multi-node + multi-cluster support
- [ ] Advanced dashboards with filtering & sorting
- [ ] Integration with OpenTelemetry (OTel)
- [ ] Observability for CI/CD, cloud, and cost

---

## 🤝 Contributing

Contributions are welcome! Please open an issue or submit a PR if you'd like to extend Claritty.

---

## 📜 License

[MIT LICENSE](/LICENSE)