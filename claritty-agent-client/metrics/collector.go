// Updated metrics/collector.go using client-go for node info
package metrics

import (
	"context"
	"log"
	"os"
	"strconv"
	// "strings"
	"time"

	"github.com/Vaishnav88sk/claritty/claritty-agent-client/types"
	// "github.com/Vaishnav88sk/claritty/claritty-agent-client/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsapi "k8s.io/metrics/pkg/client/clientset/versioned"
)

func CollectNodeMetrics() types.Metrics {
	// Load in-cluster config
	rc, err := rest.InClusterConfig()
	if err != nil {
		log.Println("InClusterConfig error:", err)
		return types.Metrics{}
	}

	// clientset, err := kubernetes.NewForConfig(rc)
	clientset, err := kubernetes.NewForConfig(rc)
	_ = clientset // suppress unused warning for now
	if err != nil {
		log.Println("Kubernetes client error:", err)
		return types.Metrics{}
	}

	metricsClient, err := metricsapi.NewForConfig(rc)
	if err != nil {
		log.Println("Metrics client error:", err)
		return types.Metrics{}
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Println("Hostname fetch error:", err)
		return types.Metrics{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, hostname, metav1.GetOptions{})
	if err != nil {
		log.Println("Node metrics error:", err)
		return types.Metrics{}
	}

	// cpuNanoStr := nodeMetrics.Usage.Cpu().AsDec().String()
	// memBytes := nodeMetrics.Usage.Memory().Value()

	// cpuFloat := 0.0
	// if strings.Contains(cpuNanoStr, "n") {
	// 	clean := strings.TrimSuffix(cpuNanoStr, "n")
	// 	if val, err := strconv.ParseFloat(clean, 64); err == nil {
	// 		cpuFloat = val / 1e9
	// 	}
	// }
	cpuFloat, err := strconv.ParseFloat(nodeMetrics.Usage.Cpu().AsDec().String(), 64)
	if err != nil {
		log.Println("Failed to parse CPU:", err)
		cpuFloat = 0
	}

	memBytes := nodeMetrics.Usage.Memory().Value()

	return types.Metrics{
		CPU:    cpuFloat,
		Memory: int(memBytes / 1024 / 1024),
		Node:   hostname,
	}
}
