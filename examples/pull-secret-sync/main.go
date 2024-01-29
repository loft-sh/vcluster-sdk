package main

import (
	"github.com/loft-sh/vcluster-pull-secret-sync/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"k8s.io/klog/v2"
)

type PluginConfig struct {
	DestinationNamespace string `json:"destinationNamespace,omitempty"`
}

func main() {
	// Init plugin
	ctx := plugin.MustInit()

	// parse plugin config
	pluginConfig := &PluginConfig{}
	err := plugin.UnmarshalConfig(pluginConfig)
	if err != nil {
		klog.Fatal("Error parsing plugin config")
	} else if pluginConfig.DestinationNamespace == "" {
		klog.Fatal("destinationNamespace is empty")
	}

	// register syncer
	plugin.MustRegister(syncers.NewPullSecretSyncer(ctx, pluginConfig.DestinationNamespace))

	// start plugin
	plugin.MustStart()
}
