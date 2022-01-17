package main

import (
	"github.com/loft-sh/vcluster-example/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	registerContext, err := plugin.CreateContext(plugin.Options{})
	if err != nil {
		panic(err)
	}

	err = plugin.Register(syncers.NewCarSyncer(registerContext))
	if err != nil {
		panic(err)
	}

	err = plugin.Start()
	if err != nil {
		panic(err)
	}
}
