package provider

import (
	"cloud-node-lifecycle-controller/pkg/provider/aws"
	"cloud-node-lifecycle-controller/pkg/provider/azure"
	"cloud-node-lifecycle-controller/pkg/provider/tencentcloud"
	v1 "k8s.io/api/core/v1"
)

// InitCloudProvider cloud provider init function
type InitCloudProvider func() (CloudAPI, error)

// DefaultInitFuncConstructors cloud provider init function map
var DefaultInitFuncConstructors = map[string]InitCloudProvider{
	"aws": func() (CloudAPI, error) {
		return aws.InitAwsCloudProvider()
	},
	"tencent": func() (CloudAPI, error) {
		return tencentcloud.InitTencentCloudProvider()
	},
	"azure": func() (CloudAPI, error) {
		return azure.InitAzureProvider()
	},
}

// CloudAPI cloud provider interface
type CloudAPI interface {
	CheckNodeInstanceExists(node *v1.Node) (bool, error)
}
