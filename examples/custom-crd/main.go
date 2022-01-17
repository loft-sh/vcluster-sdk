package main

import "github.com/loft-sh/vcluster-sdk/plugin"

func main() {

	plugin.Register(&syncers.CarSyncer{})
}
