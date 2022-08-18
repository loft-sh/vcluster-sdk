package main

import (
	"os"

	"github.com/loft-sh/vcluster-pull-secret-sync/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

const (
	DefaultDestinationNamespace = "default"
	DestinationNamespaceEnvVar  = "DESTINATION_NAMESPACE"
)

func main() {
	// resolve configuration from environment variables
	destinationNamespace := os.Getenv(DestinationNamespaceEnvVar)
	if destinationNamespace == "" {
		destinationNamespace = DefaultDestinationNamespace
	}

	ctx := plugin.MustInit()
	plugin.MustRegister(syncers.NewPullSecretSyncer(ctx, destinationNamespace))
	plugin.MustStart()
}
