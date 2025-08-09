# cloud-node-lifecycle-controller

## description
If you use k8s on AWS/Azure/Tencent and use cluster-autoscaler, and you found the node join into cluster cannot be delete when the EC2/VM/CVM has been deleted, you can use the cloud-node-lifecycle-controller to delete the node in your cluster automatically

## Requirement
Your node created by cluster-autoscaler need have providerID :
**AWS**:   **aws:///us-west-2/abcd**             

**Azure**: **azure:///subscriptions/sub123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm-01**

**Tencent**:   **qcloud:///ap-singapore/ins-abcd**


## Usage
```shell
cloud-node-lifecycle-controller \
--log_dir=/usr/local/fountain/cloud-node-lifecycle-controller/logs  \
--logtostderr=false  \
--cloud-provider=aws  \
--region=us-west-2  \
--access-key-id=xxxx  \
--secret-key-id=yyyy  \
--port=8080
```

## Development
If you want to extend the controller on other cloud
1. Correctly set the providerID on node created by cluster-autoscaler according to the cloud specifications.
2. add a new directory in pkg/provider and add a go file in it
3. Implement the interface CloudAPI and override CheckNodeInstanceExists method to check if the node can be deleted
4. add the registration method and add it in pkg/provider/cloud-provider.go
5. try it!