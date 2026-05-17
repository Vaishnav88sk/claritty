// Package k8s collects cluster state using the Kubernetes API.
// It uses InClusterConfig when running inside a pod, or falls back to
// kubeconfig for local development.
package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Vaishnav88sk/claritty/sre-agent/agent/internal/incident"
)

// Client wraps the Kubernetes clientset.
type Client struct {
	cs kubernetes.Interface
}

// PodInfo is a lightweight description of a pod's state.
type PodInfo struct {
	Name      string
	Namespace string
	Phase     string
	Restarts  int32
	Ready     bool
	Node      string
	Reason    string // CrashLoopBackOff, ImagePullBackOff, OOMKilled, etc.
	Message   string
	Logs      string // last 50 lines of logs
}

// New creates a Kubernetes client. Prefers in-cluster config, then kubeconfig.
func New() (*Client, error) {
	cfg, err := buildConfig()
	if err != nil {
		return nil, fmt.Errorf("k8s config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s client: %w", err)
	}
	return &Client{cs: cs}, nil
}

func buildConfig() (*rest.Config, error) {
	// 1. In-cluster (running inside a pod — production)
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	// 2. KUBECONFIG env var
	if kc := os.Getenv("KUBECONFIG"); kc != "" {
		return clientcmd.BuildConfigFromFlags("", kc)
	}
	// 3. Default ~/.kube/config (local development)
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", defaultPath)
	}
	return nil, fmt.Errorf("no kubeconfig found and not running in-cluster")
}

// CollectSnapshot gathers a full cluster health snapshot and all pod states.
func (c *Client) CollectSnapshot(ctx context.Context, namespaces []string) (*incident.ClusterSnapshot, []PodInfo, error) {
	// 1. Nodes
	nodeList, err := c.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("list nodes: %w", err)
	}

	snap := &incident.ClusterSnapshot{}
	snap.TotalNodes = len(nodeList.Items)
	nsSet := map[string]bool{}

	for _, n := range nodeList.Items {
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				snap.ReadyNodes++
			}
		}
	}

	// 2. Pods across namespaces (or all)
	var targetNS []string
	if len(namespaces) == 0 {
		nsList, err := c.cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, nil, fmt.Errorf("list namespaces: %w", err)
		}
		for _, ns := range nsList.Items {
			targetNS = append(targetNS, ns.Name)
		}
	} else {
		targetNS = namespaces
	}

	var pods []PodInfo
	for _, ns := range targetNS {
		podList, err := c.cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		nsSet[ns] = true
		for _, p := range podList.Items {
			info := summarizePod(p)
			pods = append(pods, info)

			switch strings.ToLower(string(p.Status.Phase)) {
			case "running":
				snap.RunningPods++
			case "pending":
				snap.PendingPods++
			case "failed":
				snap.FailedPods++
			}
			if isCrashLoop(info.Reason) {
				snap.CrashloopPods++
			}
		}
	}

	for ns := range nsSet {
		snap.Namespaces = append(snap.Namespaces, ns)
	}

	// 3. Fetch logs for problematic pods only (to keep API load minimal)
	for i := range pods {
		if isProblematic(pods[i]) {
			pods[i].Logs = c.fetchLogs(ctx, pods[i].Namespace, pods[i].Name)
		}
	}

	// 4. Compute health score
	snap.HealthScore = computeHealthScore(snap)

	return snap, pods, nil
}

// CollectEvents returns recent warning events, scoped to a namespace or all.
func (c *Client) CollectEvents(ctx context.Context, namespace string) string {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	evList, err := c.cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         50,
	})
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, ev := range evList.Items {
		msg := ev.Message
		if len(msg) > 200 {
			msg = msg[:200]
		}
		fmt.Fprintf(&sb, "[%s] %s/%s: %s — %s\n",
			ev.InvolvedObject.Kind, ev.InvolvedObject.Namespace,
			ev.InvolvedObject.Name, ev.Reason, msg)
	}
	return sb.String()
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func summarizePod(p corev1.Pod) PodInfo {
	info := PodInfo{
		Name:      p.Name,
		Namespace: p.Namespace,
		Phase:     string(p.Status.Phase),
		Node:      p.Spec.NodeName,
	}
	for _, cs := range p.Status.ContainerStatuses {
		info.Restarts += cs.RestartCount
		if cs.Ready {
			info.Ready = true
		}
		if cs.State.Waiting != nil {
			info.Reason = cs.State.Waiting.Reason
			info.Message = cs.State.Waiting.Message
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			info.Reason = cs.State.Terminated.Reason
		}
	}
	return info
}

func (c *Client) fetchLogs(ctx context.Context, ns, name string) string {
	tailLines := int64(50)
	req := c.cs.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{
		TailLines: &tailLines,
	})
	raw, err := req.DoRaw(ctx)
	if err != nil {
		return ""
	}
	logs := string(raw)
	if len(logs) > 2000 {
		logs = logs[len(logs)-2000:]
	}
	return logs
}

func isProblematic(p PodInfo) bool {
	return p.Restarts > 2 || isCrashLoop(p.Reason) || p.Phase == "Pending" || p.Phase == "Failed"
}

func isCrashLoop(reason string) bool {
	return reason == "CrashLoopBackOff" || reason == "OOMKilled"
}

func computeHealthScore(snap *incident.ClusterSnapshot) float64 {
	score := 100.0
	total := snap.RunningPods + snap.PendingPods + snap.FailedPods
	if total > 0 {
		badFraction := float64(snap.PendingPods+snap.FailedPods) / float64(total)
		score -= badFraction * 40
	}
	score -= float64(snap.CrashloopPods) * 5
	if snap.TotalNodes > 0 && snap.ReadyNodes < snap.TotalNodes {
		score -= float64(snap.TotalNodes-snap.ReadyNodes) * 20
	}
	if score < 0 {
		score = 0
	}
	return score
}
