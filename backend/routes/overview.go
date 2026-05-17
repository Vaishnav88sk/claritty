package routes

import (
	"context"
	"net/http"
	"time"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsapi "k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetClusterOverview(c *gin.Context) {
	// In-cluster config
	// config, err := rest.InClusterConfig()
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println("InClusterConfig failed, using kubeconfig...")

		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Println("Kubeconfig failed:", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	clientset, _ := kubernetes.NewForConfig(config)
	metricsClient, _ := metricsapi.NewForConfig(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔹 Nodes
	nodes, _ := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	totalNodes := len(nodes.Items)
	readyNodes := 0

	var totalCPU float64 = 0
	var totalMemory float64 = 0

	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				readyNodes++
			}
		}

		cpu := node.Status.Capacity.Cpu().MilliValue()
		mem := node.Status.Capacity.Memory().Value()

		totalCPU += float64(cpu) / 1000
		totalMemory += float64(mem) / (1024 * 1024)
	}

	// 🔹 Node usage
	nodeMetrics, _ := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})

	var usedCPU float64 = 0
	var usedMemory float64 = 0

	for _, m := range nodeMetrics.Items {
		usedCPU += float64(m.Usage.Cpu().MilliValue()) / 1000
		usedMemory += float64(m.Usage.Memory().Value()) / (1024 * 1024)
	}

	// 🔹 Pods
	pods, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})

	totalPods := len(pods.Items)
	runningPods := 0

	for _, p := range pods.Items {
		if p.Status.Phase == "Running" {
			runningPods++
		}
	}

	// ✅ Response (matches your UI)
	c.JSON(http.StatusOK, gin.H{
		"nodes": gin.H{
			"ready": readyNodes,
			"total": totalNodes,
		},
		"pods": gin.H{
			"running": runningPods,
			"total": totalPods,
		},
		"resources": gin.H{
			"cpu": gin.H{
				"usage": usedCPU,
				"total": totalCPU,
			},
			"memory": gin.H{
				"usage": usedMemory,
				"total": totalMemory,
			},
		},
		// optional (for now same as cluster)
		"podResources": gin.H{
			"cpu": gin.H{
				"usage": usedCPU,
				"total": totalCPU,
			},
			"memory": gin.H{
				"usage": usedMemory,
				"total": totalMemory,
			},
		},
	})
}