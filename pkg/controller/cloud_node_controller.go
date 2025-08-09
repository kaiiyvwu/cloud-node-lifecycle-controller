package controller

import (
	"cloud-node-lifecycle-controller/pkg/client"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	cloudnodeutil "k8s.io/cloud-provider/node/helpers"
	"k8s.io/klog/v2"
	"time"
)

// Controller is buffer-pool-controller struct
type Controller struct {
	ctx       context.Context
	clientset *kubernetes.Clientset

	queue workqueue.TypedRateLimitingInterface[string]

	nodeInformer cache.SharedIndexInformer
	nodeLister   listerv1.NodeLister
}

// CreateAndStartController create and start controller
func CreateAndStartController(ctx context.Context, clientset *kubernetes.Clientset, stopCh chan struct{}) {

	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]())

	controller := &Controller{
		ctx:       ctx,
		clientset: clientset,
		queue:     queue,
	}

	factory := informers.NewSharedInformerFactory(clientset, 0)

	controller.nodeInformer = factory.Core().V1().Nodes().Informer()
	controller.nodeLister = factory.Core().V1().Nodes().Lister()
	_, err := controller.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old interface{}, new interface{}) {
			node := new.(*corev1.Node)
			queue.Add(node.Name)
		},
		AddFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			queue.Add(node.Name)
		},
	})
	if err != nil {
		klog.Fatalf("failed to add event handler: %v", err)
		return
	}

	factory.Start(stopCh)

	go controller.run(stopCh)
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}

	err := func(obj string) error {
		defer c.queue.Done(obj)

		if err := c.syncHandler(obj); err != nil {
			c.queue.AddRateLimited(obj)
			return fmt.Errorf("error syncing '%s': %s, requeuing", obj, err)
		}

		c.queue.Forget(obj)
		return nil
	}(key)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// Run start to watch and handler
func (c *Controller) run(stopCh chan struct{}) {
	defer runtime.HandleCrash()

	defer c.queue.ShutDown()

	if !cache.WaitForCacheSync(stopCh, c.nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < 5; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			nodeList, err := c.clientset.CoreV1().Nodes().List(c.ctx, metav1.ListOptions{})
			if err != nil {
				klog.Errorf("list node error:%v", err)
				return
			}
			for _, node := range nodeList.Items {
				err := c.processNode(&node)
				if err != nil {
					klog.Errorf("process node %s error:%v", node.Name, err)
					return
				}
			}
		}
	}()

	<-stopCh
	klog.Info("Stopping Cloud Node Controller")
}

func (c *Controller) syncHandler(nodeName string) error {
	node, err := c.nodeLister.Get(nodeName)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("get node %s error: %v", nodeName, err)
			return err
		} else {
			klog.Infof("node %s is not found", nodeName)
			return nil
		}

	}
	klog.Infof("syncHandler nodeName: %s", nodeName)
	if err = c.processNode(node); err != nil {
		return err
	}
	return nil
}

func (c *Controller) processNode(node *corev1.Node) error {
	nodeName := node.Name

	status := corev1.ConditionUnknown
	if _, c := cloudnodeutil.GetNodeCondition(&node.Status, corev1.NodeReady); c != nil {
		status = c.Status
	}

	if status != corev1.ConditionTrue && node.Spec.ProviderID != "" {
		klog.Infof("node %s is not ready, try to check machine status", nodeName)
		existed, err := client.CloudProviderAPI.CheckNodeInstanceExists(node)
		if err != nil {
			return err
		}
		if !existed {
			klog.Infof("node %s is not existed on cloud,will delete it", nodeName)
			if err := c.clientset.CoreV1().Nodes().Delete(context.TODO(), nodeName, metav1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					klog.Errorf("delete node %s error: %v", nodeName, err)
					return err
				} else {
					klog.Infof("node %s is not found", nodeName)
					return nil
				}

			}
			klog.Infof("delete node %s success", nodeName)
		}
	}
	return nil
}
