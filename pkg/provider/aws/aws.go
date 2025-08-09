package aws

import (
	"cloud-node-lifecycle-controller/pkg/config"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"strings"
)

// Aws aws cloud provider
type Aws struct {
}

// InitAwsCloudProvider init aws cloud provider
func InitAwsCloudProvider() (*Aws, error) {
	return &Aws{}, nil
}

// parseInstanceFromProviderID parse instance id from provider id
func parseInstanceFromProviderID(node *v1.Node) (string, string, error) {
	providerID := node.Spec.ProviderID
	metadata := strings.Split(strings.TrimPrefix(providerID, "aws://"), "/")
	klog.Infof("node %s providerID %s", node.Name, providerID)
	if len(metadata) == 3 {
		return metadata[1], metadata[2], nil
	}
	return "", "", fmt.Errorf("invalid providerID: %s", providerID)
}

// CheckNodeInstanceExists check node instance exists
func (a *Aws) CheckNodeInstanceExists(node *v1.Node) (bool, error) {
	providerID := node.Spec.ProviderID
	_, instanceID, err := parseInstanceFromProviderID(node)
	region := config.Options.Region
	if err != nil {
		klog.Errorf("Failed to parse instance ID from provider ID %s: %v", providerID, err)
		return false, err
	}
	klog.Infof("region: %s, instanceID: %s", region, instanceID)
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(config.Options.AccessKeyID, config.Options.SecretKeyID, ""),
	})
	if err != nil {
		klog.Fatalf("Failed to create session: %v", err)
	}

	svc := ec2.New(sess)

	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{instanceID}),
	})
	if err != nil {

		if awsError, ok := err.(awserr.Error); ok {
			if awsError.Code() == ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound {
				klog.Infof("Instance %s not found.", instanceID)
				return false, nil
			}
			klog.Errorf("Failed to describe  %s: %v", instanceID, err)
			return true, err
		} else if strings.Contains(err.Error(), ec2.UnsuccessfulInstanceCreditSpecificationErrorCodeInvalidInstanceIdNotFound) {
			klog.Infof("Instance %s not found.", instanceID)
			return false, nil
		}
		klog.Errorf("Failed to describe  %s: %v", instanceID, err)
		return true, err

	}

	if len(resp.Reservations) == 0 {
		klog.Infof("Instance %s not found.\n", instanceID)
		return false, nil
	}

	instance := resp.Reservations[0].Instances[0]
	state := *instance.State.Name

	klog.Infof("Instance %s state: %s", instanceID, state)

	return state != ec2.InstanceStateNameTerminated && state != ec2.InstanceStateNameShuttingDown, nil
}
