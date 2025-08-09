package client

import (
	"cloud-node-lifecycle-controller/pkg/provider"
	"k8s.io/client-go/kubernetes"
)

// Client client for server
// CloudProviderApi cloud provider for server
var (
	Client           *kubernetes.Clientset //Client for server
	CloudProviderAPI provider.CloudAPI     //CloudProviderApi cloud provider for server
)
