package azure

import (
	"cloud-node-lifecycle-controller/pkg/config"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Azure is a provider that checks for VM existence in Azure.
type Azure struct {
	vmClient *armcompute.VirtualMachinesClient
}

// NewProvider creates a new Azure provider using Managed Identity to authenticate.
// It acquires credentials via DefaultAzureCredential, which supports Managed Identity.
func InitAzureProvider() (*Azure, error) {
	// Use DefaultAzureCredential, which will use Managed Identity if available
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire Azure credential: %w", err)
	}
	// Create the VM client
	client, err := armcompute.NewVirtualMachinesClient(config.Options.SubscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create VirtualMachinesClient: %w", err)
	}
	return &Azure{client}, nil
}

// parseInstanceFromProviderID parse resource group and VM name from provider id
func parseInstanceFromProviderID(node *corev1.Node) (string, string, error) {
	// providerid azure:///subscriptions/sub123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm-01
	providerID := node.Spec.ProviderID
	if !strings.HasPrefix(providerID, "azure://") {
		return "", "", fmt.Errorf("invalid providerID: %s", providerID)
	}

	// split providerid
	path := strings.TrimPrefix(providerID, "azure://")
	// parts: ["subscriptions","<sub>","resourceGroups","<rg>","providers","Microsoft.Compute","virtualMachines","<vm>"]
	parts := strings.Split(path, "/")

	// find "resourceGroups" and "virtualMachines" positions
	if len(parts) == 8 && parts[2] == "resourceGroups" && parts[7] == "virtualMachines" {
		resourceGroup := parts[3]
		vmName := parts[7]
		return resourceGroup, vmName, nil
	}

	return "", "", fmt.Errorf("invalid providerID format: %s", providerID)
}

// CheckNodeInstanceExists check if the Azure VM instance exists
func (a *Azure) CheckNodeInstanceExists(node *corev1.Node) (bool, error) {
	resourceGroup, vmName, err := parseInstanceFromProviderID(node)
	if err != nil {
		klog.Errorf("Failed to parse instance ID from provider ID %s: %v", node.Spec.ProviderID, err)
		return false, err
	}

	ctx := context.Background()
	opts := &armcompute.VirtualMachinesClientGetOptions{
		Expand: to.Ptr(armcompute.InstanceViewTypesInstanceView),
	}
	_, err = a.vmClient.Get(ctx, resourceGroup, vmName, opts)
	if err != nil {
		if isNotFoundError(err) {
			klog.Infof("Instance %s not found, has been released.", vmName)
			return false, nil
		}
		klog.Errorf("Failed to get VM %s: %v", vmName, err)
		return true, err
	}
	return true, nil
}

// isNotFoundError returns true if the error is a 404 Not Found from Azure.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// SDK returns *azcore.ResponseError for HTTP errors
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}
