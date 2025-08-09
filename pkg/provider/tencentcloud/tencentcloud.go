package tencentcloud

import (
	"cloud-node-lifecycle-controller/pkg/config"
	"fmt"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"strings"
)

// Tencent tencent cloud provider
type Tencent struct {
	client *cvm.Client
}

// parseInstanceFromProviderID parse instance id from provider id
func parseInstanceFromProviderID(node *v1.Node) (string, string, error) {
	providerID := node.Spec.ProviderID
	metadata := strings.Split(strings.TrimPrefix(providerID, "qcloud:///"), "/")
	klog.Infof("node %s providerID %s", node.Name, providerID)
	if len(metadata) == 2 {
		return metadata[0], metadata[1], nil
	}
	return "", "", fmt.Errorf("invalid providerID: %s", providerID)
}

// InitTencentCloudProvider init tencent cloud provider
func InitTencentCloudProvider() (*Tencent, error) {
	credential := common.NewCredential(config.Options.AccessKeyID, config.Options.SecretKeyID)
	// 设置客户端配置
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"

	// 初始化客户端
	client, err := cvm.NewClient(credential, config.Options.Region, cpf)

	if err != nil {
		panic(fmt.Sprintf("创建客户端失败: %v", err))
	}

	return &Tencent{client}, nil
}

// CheckNodeInstanceExists check node instance exists
func (t *Tencent) CheckNodeInstanceExists(node *v1.Node) (bool, error) {
	providerID := node.Spec.ProviderID
	_, instanceID, err := parseInstanceFromProviderID(node)
	region := config.Options.Region
	if err != nil {
		klog.Errorf("Failed to parse instance ID from provider ID %s: %v", providerID, err)
		return false, err
	}
	klog.Infof("region: %s, instanceID: %s", region, instanceID)
	// 创建请求并设置实例ID
	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{instanceID})
	resp, err := t.client.DescribeInstances(request)
	if err != nil {
		klog.Errorf("Failed to describe  %s: %v", instanceID, err)
		return true, err
	}
	if len(resp.Response.InstanceSet) == 0 {
		klog.Infof("Instance %s not found, has been released.\n", instanceID)
		return false, nil
	}
	instance := resp.Response.InstanceSet[0]
	state := *instance.InstanceState

	klog.Infof("Instance %s state: %s", instanceID, state)
	return state != "TERMINATING", nil
}
