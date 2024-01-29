package main

import (
	"github.com/loft-sh/vcluster-pull-secret-sync/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := plugin.MustInit()
	plugin.MustRegister(syncers.NewPullSecretSyncer(ctx, "todo"))
	plugin.MustStart()
}
