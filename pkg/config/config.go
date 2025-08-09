package config

import (
	"cloud-node-lifecycle-controller/pkg/option"
	"context"
)

// CloudProvider cloud provider for server
var (
	Options *option.Options
	Context context.Context //context for server
)
