package main

import (
	"os"

	"github.com/loft-sh/vcluster-pull-secret-sync/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultDestinationNamespace = "default"
	LabelSelectorEnvVar         = "LABEL_SELECTOR"
	DestinationNamespaceEnvVar  = "DESTINATION_NAMESPACE"
)

func main() {
	//Resolve configuration from environment variables
	labelSelector := metav1.LabelSelector{}
	selector := os.Getenv(LabelSelectorEnvVar)
	if selector != "" {
		//TODO: parse the label selector, log and exit on error
	}
	destinationNamespace := os.Getenv(DestinationNamespaceEnvVar)
	if destinationNamespace == "" {
		destinationNamespace = DefaultDestinationNamespace
	}

	ctx := plugin.MustInit("pull-secret-sync-plugin")
	plugin.MustRegister(syncers.NewPullSecretSyncer(ctx, labelSelector, destinationNamespace))
	plugin.MustStart()
}
