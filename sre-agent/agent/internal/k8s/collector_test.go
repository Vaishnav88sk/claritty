package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCollectSnapshot_Success(t *testing.T) {
	ctx := context.Background()

	// 1. Setup mock objects
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "default"},
	}

	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
		},
	}

	healthyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "healthy-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Ready: true, RestartCount: 0},
			},
		},
	}

	crashingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "crashing-pod", Namespace: "default"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning, // CrashLoopBackOff pods technically have "Running" phase often but not ready
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready:        false,
					RestartCount: 5,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}

	// 2. Create fake client
	cs := fake.NewSimpleClientset(ns, node1, node2, healthyPod, crashingPod)
	c := NewWithClient(cs)

	// 3. Act
	snap, pods, err := c.CollectSnapshot(ctx, nil) // passing nil for namespaces triggers listing all namespaces
	if err != nil {
		t.Fatalf("CollectSnapshot failed: %v", err)
	}

	// 4. Assert Snapshot Metrics
	if snap.TotalNodes != 2 {
		t.Errorf("Expected 2 nodes, got %d", snap.TotalNodes)
	}
	if snap.ReadyNodes != 1 {
		t.Errorf("Expected 1 ready node, got %d", snap.ReadyNodes)
	}
	if snap.RunningPods != 2 {
		t.Errorf("Expected 2 running pods (based on phase), got %d", snap.RunningPods)
	}
	if snap.CrashloopPods != 1 {
		t.Errorf("Expected 1 crashloop pod, got %d", snap.CrashloopPods)
	}

	// Health score penalizes: -20 for the 1 offline node, -5 for the 1 crashloop pod
	// Total base = 100
	// Wait, bad fraction penalty: 0 pending/failed out of 2 total -> 0
	// So score = 100 - 0 - 5 - 20 = 75
	if snap.HealthScore != 75 {
		t.Errorf("Expected health score 75, got %.2f", snap.HealthScore)
	}

	// 5. Assert PodInfo Mapping
	if len(pods) != 2 {
		t.Fatalf("Expected 2 pods in list, got %d", len(pods))
	}

	var foundCrash bool
	for _, p := range pods {
		if p.Name == "crashing-pod" {
			foundCrash = true
			if p.Reason != "CrashLoopBackOff" {
				t.Errorf("Expected crashing-pod reason to be CrashLoopBackOff, got %s", p.Reason)
			}
			if p.Restarts != 5 {
				t.Errorf("Expected crashing-pod to have 5 restarts, got %d", p.Restarts)
			}
		}
	}
	if !foundCrash {
		t.Errorf("Expected to find crashing-pod in returned slice")
	}
}

func TestCollectEvents_WarningFilter(t *testing.T) {
	ctx := context.Background()

	// The client-go fake clientset does not execute FieldSelectors natively.
	// We just inject a Warning event and ensure it gets formatted correctly by CollectEvents.
	warningEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "event-2", Namespace: "default"},
		Type:       "Warning",
		Reason:     "FailedScheduling",
		Message:    "0/1 nodes are available",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "test-pod",
		},
	}

	cs := fake.NewSimpleClientset(warningEvent)
	c := NewWithClient(cs)

	eventsString := c.CollectEvents(ctx, "default")

	if !strings.Contains(eventsString, "FailedScheduling") || !strings.Contains(eventsString, "0/1 nodes are available") {
		t.Errorf("Expected output to contain Warning events, got: %s", eventsString)
	}
}

func TestCollectSnapshot_NamespaceFilter(t *testing.T) {
	ctx := context.Background()

	podDefault := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-default", Namespace: "default"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	podKubeSystem := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod-kube-system", Namespace: "kube-system"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}

	cs := fake.NewSimpleClientset(podDefault, podKubeSystem)
	c := NewWithClient(cs)

	// Filter only "default" namespace
	_, pods, err := c.CollectSnapshot(ctx, []string{"default"})
	if err != nil {
		t.Fatalf("CollectSnapshot failed: %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("Expected 1 pod returned, got %d", len(pods))
	}
	if pods[0].Name != "pod-default" {
		t.Errorf("Expected pod-default, got %s", pods[0].Name)
	}
}
