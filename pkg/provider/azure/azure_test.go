package azure

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

// --- Tests for parseInstanceFromProviderID ---

func TestParseInstanceFromProviderID_Success(t *testing.T) {
	// 典型 Azure providerID，去掉多余的 '/'
	node := &corev1.Node{}
	node.Spec.ProviderID = "azure:///subscriptions/sub123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm-01"

	rg, vm, err := parseInstanceFromProviderID(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rg != "rg1" {
		t.Errorf("expected resourceGroup 'rg1', got '%s'", rg)
	}
	if vm != "vm-01" {
		t.Errorf("expected vmName 'vm-01', got '%s'", vm)
	}
}

func TestParseInstanceFromProviderID_InvalidPrefix(t *testing.T) {
	node := &corev1.Node{}
	node.Spec.ProviderID = "aws:///subscriptions/xxx"

	_, _, err := parseInstanceFromProviderID(node)
	if err == nil {
		t.Fatal("expected error for invalid prefix, got nil")
	}
}

func TestParseInstanceFromProviderID_BadFormat(t *testing.T) {
	// 缺少必要字段
	node := &corev1.Node{}
	node.Spec.ProviderID = "azure:///subscriptions/sub123/resourceGroups/rg1/virtualMachines"

	_, _, err := parseInstanceFromProviderID(node)
	if err == nil {
		t.Fatal("expected error for bad format, got nil")
	}
}

// --- Test for CheckNodeInstanceExists error branch when parsing fails ---

func TestCheckNodeInstanceExists_ParseError(t *testing.T) {
	a := &Azure{} // vmClient 不会被调用，因为 parseInstance 先返回错误
	node := &corev1.Node{}
	node.Spec.ProviderID = "invalid://id"

	exists, err := a.CheckNodeInstanceExists(node)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if exists {
		t.Errorf("expected exists=false on parse error, got true")
	}
}
