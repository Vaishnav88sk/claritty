// Package k8s wraps the official Kubernetes client-go library.
package k8s

import (
	"bytes"
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
	"k8s.io/client-go/tools/remotecommand"
)

// Client wraps a Kubernetes clientset for cluster operations.
type Client struct {
	cs  kubernetes.Interface
	cfg *rest.Config
}

// New creates a new K8s client, preferring kubeconfig > in-cluster.
func New(kubeconfigPath string) (*Client, error) {
	cfg, err := buildConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("k8s config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s client: %w", err)
	}
	return &Client{cs: cs, cfg: cfg}, nil
}

func buildConfig(overridePath string) (*rest.Config, error) {
	if overridePath != "" {
		return clientcmd.BuildConfigFromFlags("", overridePath)
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	home, _ := os.UserHomeDir()
	defaultPath := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", defaultPath)
	}
	// Fall back to in-cluster config (when running inside a pod)
	return rest.InClusterConfig()
}

// ─── Pod Operations ──────────────────────────────────────────────────────────

// PodSummary is a lightweight struct describing a pod's state.
type PodSummary struct {
	Name      string
	Namespace string
	Phase     string
	Restarts  int32
	Ready     bool
	Message   string
	Node      string
}

// ListPods returns all pods in the given namespaces.
func (c *Client) ListPods(ctx context.Context, namespaces []string) ([]PodSummary, error) {
	var out []PodSummary
	for _, ns := range namespaces {
		list, err := c.cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("list pods in %s: %w", ns, err)
		}
		for _, p := range list.Items {
			out = append(out, summarisePod(p))
		}
	}
	return out, nil
}

// DescribePod returns the raw describe output for a pod (equivalent to kubectl describe).
func (c *Client) DescribePod(ctx context.Context, ns, name string) (string, error) {
	pod, err := c.cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Name: %s\nNamespace: %s\nNode: %s\n", pod.Name, pod.Namespace, pod.Spec.NodeName)
	fmt.Fprintf(&sb, "Status: %s\n", pod.Status.Phase)
	for _, cs := range pod.Status.ContainerStatuses {
		fmt.Fprintf(&sb, "Container %s: Ready=%v Restarts=%d\n", cs.Name, cs.Ready, cs.RestartCount)
		if cs.State.Waiting != nil {
			fmt.Fprintf(&sb, "  Waiting: %s - %s\n", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
		if cs.State.Terminated != nil {
			fmt.Fprintf(&sb, "  Terminated: ExitCode=%d Reason=%s\n",
				cs.State.Terminated.ExitCode, cs.State.Terminated.Reason)
		}
	}
	for _, cond := range pod.Status.Conditions {
		fmt.Fprintf(&sb, "Condition %s: %s\n", cond.Type, cond.Status)
	}
	return sb.String(), nil
}

// GetPodLogs fetches recent logs from a pod container.
func (c *Client) GetPodLogs(ctx context.Context, ns, name, container string, tailLines int64, previous bool) (string, error) {
	req := c.cs.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
		Previous:  previous,
	})
	rc, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(rc)
	return buf.String(), err
}

// ─── Events ──────────────────────────────────────────────────────────────────

// EventSummary is a lightweight description of a Kubernetes event.
type EventSummary struct {
	Namespace string
	Type      string // "Warning" / "Normal"
	Reason    string
	Object    string
	Message   string
	Count     int32
	Age       string
}

// GetWarningEvents returns Warning events across the given namespaces.
func (c *Client) GetWarningEvents(ctx context.Context, namespaces []string, maxCount int) ([]EventSummary, error) {
	var out []EventSummary
	for _, ns := range namespaces {
		list, err := c.cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{
			FieldSelector: "type=Warning",
		})
		if err != nil {
			return nil, err
		}
		for i, ev := range list.Items {
			if i >= maxCount {
				break
			}
			msg := ev.Message
			if len(msg) > 200 {
				msg = msg[:200] + "..."
			}
			out = append(out, EventSummary{
				Namespace: ev.Namespace,
				Type:      ev.Type,
				Reason:    ev.Reason,
				Object:    fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
				Message:   msg,
				Count:     ev.Count,
			})
		}
	}
	return out, nil
}

// ─── Nodes ───────────────────────────────────────────────────────────────────

// NodeSummary is a lightweight snapshot of a node's health.
type NodeSummary struct {
	Name       string
	Ready      bool
	Cordoned   bool
	Roles      []string
	Conditions map[string]string
}

// GetNodeHealth returns health info for all nodes.
func (c *Client) GetNodeHealth(ctx context.Context) ([]NodeSummary, error) {
	list, err := c.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var out []NodeSummary
	for _, n := range list.Items {
		ns := NodeSummary{
			Name:       n.Name,
			Cordoned:   n.Spec.Unschedulable,
			Conditions: make(map[string]string),
		}
		for _, cond := range n.Status.Conditions {
			ns.Conditions[string(cond.Type)] = string(cond.Status)
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ns.Ready = true
			}
		}
		out = append(out, ns)
	}
	return out, nil
}

// ─── Namespace Summary ────────────────────────────────────────────────────────

// NamespaceSummary counts pods by phase in a namespace.
type NamespaceSummary struct {
	Namespace string
	Running   int
	Pending   int
	Failed    int
	CrashLoop int
}

// GetNamespaceSummaries summarises pod counts per namespace.
func (c *Client) GetNamespaceSummaries(ctx context.Context, namespaces []string) ([]NamespaceSummary, error) {
	var out []NamespaceSummary
	for _, ns := range namespaces {
		pods, err := c.ListPods(ctx, []string{ns})
		if err != nil {
			continue
		}
		sum := NamespaceSummary{Namespace: ns}
		for _, p := range pods {
			switch p.Phase {
			case "Running":
				sum.Running++
			case "Pending":
				sum.Pending++
			case "Failed":
				sum.Failed++
			}
			if strings.Contains(p.Message, "CrashLoopBackOff") {
				sum.CrashLoop++
			}
		}
		out = append(out, sum)
	}
	return out, nil
}

// ─── Cluster Health Snapshot ─────────────────────────────────────────────────

// CollectSnapshot gathers a point-in-time cluster health summary.
func (c *Client) CollectSnapshot(ctx context.Context, namespaces []string) (*ClusterMetrics, error) {
	nodes, err := c.GetNodeHealth(ctx)
	if err != nil {
		return nil, err
	}
	pods, err := c.ListPods(ctx, namespaces)
	if err != nil {
		return nil, err
	}

	m := &ClusterMetrics{
		TotalNodes: len(nodes),
	}
	for _, n := range nodes {
		if n.Ready {
			m.ReadyNodes++
		}
	}
	m.TotalPods = len(pods)
	for _, p := range pods {
		switch p.Phase {
		case "Running":
			m.RunningPods++
		case "Pending":
			m.PendingPods++
		case "Failed":
			m.FailedPods++
		}
		if strings.Contains(p.Message, "CrashLoopBackOff") {
			m.CrashloopPods++
		}
	}
	return m, nil
}

// ClusterMetrics is the raw numbers gathered in a snapshot collection.
type ClusterMetrics struct {
	TotalNodes    int
	ReadyNodes    int
	TotalPods     int
	RunningPods   int
	PendingPods   int
	FailedPods    int
	CrashloopPods int
}

// ─── Exec ────────────────────────────────────────────────────────────────────

// ExecPod executes a command inside a running pod and returns stdout/stderr.
// This is the equivalent of kubectl exec.
func (c *Client) ExecPod(ctx context.Context, ns, pod, container string, cmd []string) (string, string, error) {
	req := c.cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(ns).
		SubResource("exec").
		Param("container", container).
		Param("stdout", "true").
		Param("stderr", "true")
	for _, c := range cmd {
		req.Param("command", c)
	}

	exec, err := remotecommand.NewSPDYExecutor(c.cfg, "POST", req.URL())
	if err != nil {
		return "", "", err
	}
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	return stdout.String(), stderr.String(), err
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func summarisePod(p corev1.Pod) PodSummary {
	ps := PodSummary{
		Name:      p.Name,
		Namespace: p.Namespace,
		Phase:     string(p.Status.Phase),
		Node:      p.Spec.NodeName,
	}
	for _, cs := range p.Status.ContainerStatuses {
		ps.Restarts += cs.RestartCount
		if cs.Ready {
			ps.Ready = true
		}
		if cs.State.Waiting != nil {
			ps.Message = cs.State.Waiting.Reason
		}
	}
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodReady {
			ps.Ready = cond.Status == corev1.ConditionTrue
		}
	}
	return ps
}
