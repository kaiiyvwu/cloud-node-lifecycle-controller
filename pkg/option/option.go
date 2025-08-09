package option

// Options struct
type Options struct {
	KubeConfig     string
	InCluster      bool
	CloudProvider  string
	Region         string
	AccessKeyID    string
	SecretKeyID    string
	Port           string
	SubscriptionID string // For Azure provider
}
