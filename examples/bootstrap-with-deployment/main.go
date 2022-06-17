package main

import (
	"github.com/loft-sh/vcluster-mydeployment-example/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := plugin.MustInit("bootstrap-with-deployment")
	plugin.MustRegister(syncers.NewMydeploymentSyncer(ctx))
	plugin.MustStart()
}
