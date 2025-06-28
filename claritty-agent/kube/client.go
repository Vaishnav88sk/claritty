package kube

import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
)

func GetClient() *kubernetes.Clientset {
    config, _ := rest.InClusterConfig()
    clientset, _ := kubernetes.NewForConfig(config)
    return clientset
}
