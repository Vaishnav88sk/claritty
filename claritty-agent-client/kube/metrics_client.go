package kube

import (
	"k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetMetricsClient() *metrics.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	metricsClient, err := metrics.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return metricsClient
}
