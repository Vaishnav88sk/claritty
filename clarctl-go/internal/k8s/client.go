// Package k8s wraps the official Kubernetes client-go library.
package k8s

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
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

// NewWithClient creates a K8s client with an existing interface, used for testing.
func NewWithClient(cs kubernetes.Interface) *Client {
	return &Client{cs: cs}
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
	Timestamp time.Time
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
			var ts time.Time
			if !ev.LastTimestamp.IsZero() {
				ts = ev.LastTimestamp.Time
			} else if !ev.EventTime.IsZero() {
				ts = ev.EventTime.Time
			} else if !ev.FirstTimestamp.IsZero() {
				ts = ev.FirstTimestamp.Time
			}
			out = append(out, EventSummary{
				Namespace: ev.Namespace,
				Type:      ev.Type,
				Reason:    ev.Reason,
				Object:    fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
				Message:   msg,
				Count:     ev.Count,
				Timestamp: ts,
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

// ─── What Changed? Investigation ─────────────────────────────────────────────

// DetectRecentChanges scans for deployments, configmaps, services, events, and node changes
// within the specified lookback window before refTime. Complies with Issue #51.
func (c *Client) DetectRecentChanges(ctx context.Context, namespaces []string, window time.Duration, refTime time.Time) ([]incident.ChangeRecord, error) {
	if window <= 0 {
		window = 60 * time.Minute
	}
	if refTime.IsZero() {
		refTime = time.Now().UTC()
	}
	windowStart := refTime.Add(-window)
	windowEnd := refTime.Add(5 * time.Minute)

	var out []incident.ChangeRecord
	seen := make(map[string]bool)

	nsList := namespaces
	if len(nsList) == 0 {
		nsList = []string{metav1.NamespaceAll}
	}

	// 1. Deployments (Rollouts, revision updates, container image updates, Git commits)
	for _, ns := range nsList {
		deps, err := c.cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, dep := range deps.Items {
				rev := dep.Annotations["deployment.kubernetes.io/revision"]
				gitCommit := dep.Annotations["git.commit"]
				if gitCommit == "" {
					gitCommit = dep.Annotations["git-commit"]
				}
				if gitCommit == "" {
					gitCommit = dep.Annotations["app.kubernetes.io/version"]
				}
				argocdWave := dep.Annotations["argocd.argoproj.io/sync-wave"]

				var img string
				if len(dep.Spec.Template.Spec.Containers) > 0 {
					img = dep.Spec.Template.Spec.Containers[0].Image
				}

				// Check condition update times (rollout updates)
				for _, cond := range dep.Status.Conditions {
					t := cond.LastUpdateTime.Time
					if !t.IsZero() && !t.Before(windowStart) && !t.After(windowEnd) {
						key := fmt.Sprintf("dep:%s:%s", dep.Namespace, dep.Name)
						if !seen[key] {
							seen[key] = true
							summary := fmt.Sprintf("%s deployed", dep.Name)
							if img != "" {
								summary = fmt.Sprintf("%s %s deployed", dep.Name, img)
							}
							if rev != "" {
								summary += fmt.Sprintf(" (rev %s)", rev)
							}

							details := map[string]interface{}{
								"revision":  rev,
								"image":     img,
								"condition": cond.Type,
								"reason":    cond.Reason,
								"message":   cond.Message,
							}
							if gitCommit != "" {
								details["git_commit"] = gitCommit
							}
							if argocdWave != "" {
								details["argocd_sync_wave"] = argocdWave
							}

							out = append(out, incident.ChangeRecord{
								Timestamp: t,
								Resource:  fmt.Sprintf("deployment/%s/%s", dep.Namespace, dep.Name),
								Namespace: dep.Namespace,
								Category:  incident.ChangeCategoryDeployment,
								Summary:   summary,
								Details:   details,
							})

							if gitCommit != "" {
								out = append(out, incident.ChangeRecord{
									Timestamp: t,
									Resource:  fmt.Sprintf("deployment/%s/%s", dep.Namespace, dep.Name),
									Namespace: dep.Namespace,
									Category:  incident.ChangeCategoryGitCommit,
									Summary:   fmt.Sprintf("Git commit %s deployed on %s", gitCommit, dep.Name),
									Details:   details,
								})
							}
						}
					}
				}
			}
		}
	}

	// 2. ConfigMaps (Configuration changes)
	for _, ns := range nsList {
		cms, err := c.cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, cm := range cms.Items {
				t := cm.CreationTimestamp.Time
				if !t.IsZero() && !t.Before(windowStart) && !t.After(windowEnd) {
					key := fmt.Sprintf("cm:%s:%s", cm.Namespace, cm.Name)
					if !seen[key] {
						seen[key] = true
						out = append(out, incident.ChangeRecord{
							Timestamp: t,
							Resource:  fmt.Sprintf("configmap/%s/%s", cm.Namespace, cm.Name),
							Namespace: cm.Namespace,
							Category:  incident.ChangeCategoryConfig,
							Summary:   fmt.Sprintf("ConfigMap %s modified/created", cm.Name),
							Details: map[string]interface{}{
								"keys_count": len(cm.Data),
							},
						})
					}
				}
			}
		}
	}

	// 3. Services (K8s Service changes)
	for _, ns := range nsList {
		svcs, err := c.cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, svc := range svcs.Items {
				t := svc.CreationTimestamp.Time
				if !t.IsZero() && !t.Before(windowStart) && !t.After(windowEnd) {
					key := fmt.Sprintf("svc:%s:%s", svc.Namespace, svc.Name)
					if !seen[key] {
						seen[key] = true
						out = append(out, incident.ChangeRecord{
							Timestamp: t,
							Resource:  fmt.Sprintf("service/%s/%s", svc.Namespace, svc.Name),
							Namespace: svc.Namespace,
							Category:  incident.ChangeCategoryKubernetes,
							Summary:   fmt.Sprintf("%s Service modified", svc.Name),
							Details: map[string]interface{}{
								"type": string(svc.Spec.Type),
							},
						})
					}
				}
			}
		}
	}

	// 4. K8s Events (Scaling, Images, ArgoCD syncs, Environment)
	for _, ns := range nsList {
		evs, err := c.cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
		if err == nil {
			for _, ev := range evs.Items {
				var t time.Time
				if !ev.LastTimestamp.IsZero() {
					t = ev.LastTimestamp.Time
				} else if !ev.EventTime.IsZero() {
					t = ev.EventTime.Time
				} else if !ev.FirstTimestamp.IsZero() {
					t = ev.FirstTimestamp.Time
				}

				if t.IsZero() || t.Before(windowStart) || t.After(windowEnd) {
					continue
				}

				key := fmt.Sprintf("ev:%s:%s:%s", ev.Namespace, ev.InvolvedObject.Name, ev.Reason)
				if seen[key] {
					continue
				}

				reason := ev.Reason
				msg := ev.Message
				lowerReason := strings.ToLower(reason)
				lowerMsg := strings.ToLower(msg)

				var cat incident.ChangeCategory
				var summary string

				if reason == "ScalingReplicaSet" || (reason == "SuccessfulCreate" && ev.InvolvedObject.Kind == "ReplicaSet") {
					cat = incident.ChangeCategoryDeployment
					summary = fmt.Sprintf("%s scaled: %s", ev.InvolvedObject.Name, msg)
				} else if reason == "Pulling" || reason == "Pulled" {
					cat = incident.ChangeCategoryImage
					summary = fmt.Sprintf("Image event for %s: %s", ev.InvolvedObject.Name, msg)
				} else if strings.Contains(lowerReason, "sync") || strings.Contains(lowerMsg, "argocd") || ev.InvolvedObject.Kind == "Application" {
					cat = incident.ChangeCategoryArgoCD
					summary = fmt.Sprintf("ArgoCD sync event on %s: %s", ev.InvolvedObject.Name, msg)
				} else if reason == "NodeNotReady" || reason == "NodeReady" {
					cat = incident.ChangeCategoryEnvironment
					summary = fmt.Sprintf("Node status changed: %s", msg)
				}

				if cat != "" {
					seen[key] = true
					out = append(out, incident.ChangeRecord{
						Timestamp: t,
						Resource:  fmt.Sprintf("%s/%s/%s", strings.ToLower(ev.InvolvedObject.Kind), ev.Namespace, ev.InvolvedObject.Name),
						Namespace: ev.Namespace,
						Category:  cat,
						Summary:   summary,
						Details: map[string]interface{}{
							"reason":  reason,
							"message": msg,
							"count":   ev.Count,
						},
					})
				}
			}
		}
	}

	return out, nil
}
