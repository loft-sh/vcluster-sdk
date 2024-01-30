package main

import (
	examplev1 "github.com/loft-sh/vcluster-sdk/e2e/test_plugin/apis/v1"
	"github.com/loft-sh/vcluster-sdk/e2e/test_plugin/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster/pkg/scheme"
)

func main() {
	_ = examplev1.AddToScheme(scheme.Scheme)

	ctx := plugin.MustInit()
	plugin.MustRegister(syncers.NewServiceHook())
	plugin.MustRegister(syncers.NewPodHook())
	plugin.MustRegister(syncers.NewSecretHook())
	plugin.MustRegister(syncers.NewMyDeploymentSyncer(ctx))
	plugin.MustRegister(syncers.NewCarSyncer(ctx))
	plugin.MustStart()
}
