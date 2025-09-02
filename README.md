# Claritty – Observability and Monitoring 

### Agent for Kubernetes clusters, CI-CD, Anamoly detection  

Claritty is a **open-source, lightweight, cloud-native observability agent** for Kubernetes clusters.  
It collects **real-time node and custom metrics** and forwards them to a backend service for visualization and analysis. Claritty is built to be **scalable, extensible, and easy to deploy** across any Kubernetes environment. Expanding into end-to-end DevOps observability & monitoring ...

---

## ✨ Features  
- 📊 **Node-level metrics**: Collects CPU, memory, and other resource usage using [`client-go`](https://github.com/kubernetes/client-go) and the Kubernetes **metrics-server**.  
- ⚡ **Custom metrics**: Fetches additional workload and API-driven metrics using the Kubernetes API.  
- 🛰 **DaemonSet deployment**: Runs on every node to ensure cluster-wide coverage.  
- 🌐 **Backend integration**: Sends metrics to a backend service (Go + Gin) hosted on EC2 for processing and dashboard visualization.  
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

### 1. Deploy Claritty Agent  
```bash
kubectl apply -f agent-daemonset.yaml
```

### 2. Backend Setup  
Run the Go backend service (on EC2 or locally):  
```bash
go run main.go
```

### 3. View Metrics  
Access the dashboard to visualize node and custom metrics.  

---

## 📈 Example Metrics Collected  
- Node name  
- CPU usage (%)  
- Memory usage (MB)  
- Node metrics via Kubernetes API  
- And many more custom metrics you want 😉

---

## 🔮 Roadmap  
- [ ] Namespace-level observability  
- [ ] Multi-node + multi-cluster support  
- [ ] Advanced dashboards with filtering & sorting  
- [ ] Integration with OpenTelemetry (OTel)  
- [ ] Observability for CI/CD, cloud, and cost
- [ ] AI-driven anomaly detection on metrics  

---

## 🤝 Contributing  
Contributions are welcome! Please open an issue or submit a PR if you’d like to extend Claritty.  

---

## 📜 License  
[MIT LICENSE](/LiCENSE)