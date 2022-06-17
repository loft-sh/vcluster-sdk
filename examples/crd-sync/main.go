package main

import (
	"github.com/loft-sh/vcluster-example/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := plugin.MustInit("crd-sync")
	plugin.MustRegister(syncers.NewCarSyncer(ctx))
	plugin.MustStart()
}
