package main

import (
	"cloud-node-lifecycle-controller/pkg/client"
	"cloud-node-lifecycle-controller/pkg/config"
	"cloud-node-lifecycle-controller/pkg/controller"
	"cloud-node-lifecycle-controller/pkg/option"
	"cloud-node-lifecycle-controller/pkg/provider"
	"cloud-node-lifecycle-controller/pkg/server"
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"os"
	"time"
)

// NAMESPACE cloud node lifecycle controller lease namespace
const (
	NAMESPACE = "kube-system"
)

var processIndentify string

func init() {
	processIndentify = os.Getenv("HOSTNAME")
	if processIndentify == "" {
		processIndentify = uuid.New().String()
	}
}

func main() {
	_ = flag.CommandLine.Parse([]string{})
	klog.InitFlags(flag.CommandLine)
	var o = option.Options{}
	stopCh := make(chan struct{})
	cmd := &cobra.Command{
		Use:   "cloud-node-lifecycle-controller",
		Short: "cloud-node-lifecycle-controller",
		Run: func(cmd *cobra.Command, args []string) {
			if o.CloudProvider == "" {
				klog.Fatalf("cloud provider can't be empty")
				return
			}
			if _, ok := provider.DefaultInitFuncConstructors[o.CloudProvider]; !ok {
				klog.Fatalf("cloud provider %s not support", o.CloudProvider)
				return
			}
			if o.Region == "" {
				klog.Fatalf("region can't be empty")
				return
			}

			api, err := provider.DefaultInitFuncConstructors[o.CloudProvider]()
			if err != nil {
				return
			}

			client.CloudProviderAPI = api
			initClusterConfig(o.InCluster, o.KubeConfig)
			go startLeaderElection(stopCh)
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.PersistentFlags().StringVar(&o.KubeConfig, "kube-config", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	cmd.PersistentFlags().BoolVar(&o.InCluster, "in-cluster", true, "If not in cluster,need to specify kubeconfig path")
	cmd.PersistentFlags().StringVar(&o.CloudProvider, "cloud-provider", "", "cloud provider, support aws azure tencent")
	cmd.PersistentFlags().StringVar(&o.SubscriptionID, "subscription-id", "", "subscription id for azure cloud provider")
	cmd.PersistentFlags().StringVar(&o.Region, "region", "", "instance region")
	cmd.PersistentFlags().StringVar(&o.AccessKeyID, "access-key-id", "", "access key id")
	cmd.PersistentFlags().StringVar(&o.SecretKeyID, "secret-key-id", "", "secret")
	cmd.PersistentFlags().StringVar(&o.Port, "port", "8080", "health check port")

	config.Options = &o

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	config.Context = ctx

	defer klog.Flush()

	defer func() {
		close(stopCh)
		cancel()
	}()
	go server.NewAPIServer(o.Port)
	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}

	select {}

}

func startLeaderElection(stop chan struct{}) {
	clientset := client.Client
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      "cloud-node-lifecycle-controller",
			Namespace: NAMESPACE,
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: processIndentify,
		},
	}
	klog.Infof("start to acquire lease")

	leaderelection.RunOrDie(context.TODO(), leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("current acquire leader success")
				controller.CreateAndStartController(ctx, clientset, stop)
			},
			OnStoppedLeading: func() {
				klog.Infof("current process:%s lost lease", processIndentify)
				klog.Flush()
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				if identity == processIndentify {
					klog.Infof("new leader is current process:%s", processIndentify)
					return
				}
				klog.Infof("new leader is :%s", identity)
			},
		},
	})
}

func initClusterConfig(inCluster bool, kubeConfig string) {
	var config *rest.Config
	var err error
	if inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		client.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			panic(err)
		}
		client.Client, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err)
		}
	}
}
