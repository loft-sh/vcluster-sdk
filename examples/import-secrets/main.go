package main

import (
	"github.com/loft-sh/vcluster-import-secrets/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	// Init plugin
	ctx := plugin.MustInit()

	// register syncer
	plugin.MustRegister(syncers.NewImportSecrets(ctx))

	// start plugin
	plugin.MustStart()
}
