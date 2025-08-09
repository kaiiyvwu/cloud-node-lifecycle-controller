package aws

import (
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAwsCheckNode(t *testing.T) {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: uuid.New().String(),
		},
		Spec: v1.NodeSpec{
			ProviderID: "aws:///us-west-2/bbb",
		},
	}
	api, err := InitAwsCloudProvider()
	if err != nil {
		return
	}
	_, err = api.CheckNodeInstanceExists(node)
	if err != nil {
		return
	}
}
