package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestClient_DetectRecentChanges(t *testing.T) {
	now := time.Now().UTC()
	window := 30 * time.Minute

	// Deployment updated 5 minutes ago
	recentDepTime := metav1.NewTime(now.Add(-5 * time.Minute))
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "payment-api",
			Namespace: "default",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "43",
				"app.kubernetes.io/version":         "v1.43",
				"git.commit":                        "7a8b9c",
				"argocd.argoproj.io/sync-wave":      "1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Image: "payment-api:v1.43"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:           appsv1.DeploymentProgressing,
					Status:         corev1.ConditionTrue,
					Reason:         "NewReplicaSetAvailable",
					LastUpdateTime: recentDepTime,
				},
			},
		},
	}

	// ConfigMap created 3 minutes ago
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "payment-config",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Minute)),
		},
		Data: map[string]string{
			"REDIS_HOST": "redis.default.svc",
		},
	}

	// Service created 4 minutes ago
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "redis",
			Namespace:         "default",
			CreationTimestamp: metav1.NewTime(now.Add(-4 * time.Minute)),
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Warning event 2 minutes ago
	ev := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ev1",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind: "ReplicaSet",
			Name: "payment-api-abc",
		},
		Reason:        "ScalingReplicaSet",
		Message:       "Scaled up replica set payment-api-abc to 3",
		LastTimestamp: metav1.NewTime(now.Add(-2 * time.Minute)),
	}

	// Old deployment updated 2 hours ago (outside window)
	oldDepTime := metav1.NewTime(now.Add(-2 * time.Hour))
	oldDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "legacy-app",
			Namespace: "default",
		},
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:           appsv1.DeploymentAvailable,
					LastUpdateTime: oldDepTime,
				},
			},
		},
	}

	fakeCS := fake.NewSimpleClientset(dep, cm, svc, ev, oldDep)
	client := NewWithClient(fakeCS)

	changes, err := client.DetectRecentChanges(context.Background(), []string{"default"}, window, now)
	if err != nil {
		t.Fatalf("DetectRecentChanges failed: %v", err)
	}

	if len(changes) == 0 {
		t.Fatalf("Expected detected changes, got 0")
	}

	foundDeployment := false
	foundGit := false
	foundConfig := false
	foundService := false
	foundOld := false

	for _, c := range changes {
		if c.Category == incident.ChangeCategoryDeployment && c.Resource == "deployment/default/payment-api" {
			foundDeployment = true
		}
		if c.Category == incident.ChangeCategoryGitCommit && c.Details["git_commit"] == "7a8b9c" {
			foundGit = true
		}
		if c.Category == incident.ChangeCategoryConfig && c.Resource == "configmap/default/payment-config" {
			foundConfig = true
		}
		if c.Category == incident.ChangeCategoryKubernetes && c.Resource == "service/default/redis" {
			foundService = true
		}
		if c.Resource == "deployment/default/legacy-app" {
			foundOld = true
		}
	}

	if !foundDeployment {
		t.Errorf("Expected to find deployment change for payment-api")
	}
	if !foundGit {
		t.Errorf("Expected to find git commit change for payment-api")
	}
	if !foundConfig {
		t.Errorf("Expected to find config change for payment-config")
	}
	if !foundService {
		t.Errorf("Expected to find service change for redis")
	}
	if foundOld {
		t.Errorf("Old deployment outside 30m window should not be included")
	}
}
