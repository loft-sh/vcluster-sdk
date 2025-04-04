package main

import (
	"fmt"
	"os"

	examplev1 "github.com/loft-sh/vcluster-sdk/e2e/test_plugin/apis/v1"
	"github.com/loft-sh/vcluster-sdk/e2e/test_plugin/syncers"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster/pkg/mappings/resources"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"k8s.io/klog/v2"
)

type PluginConfig struct {
	String string            `json:"string,omitempty"`
	Int    int               `json:"int,omitempty"`
	Map    map[string]string `json:"map,omitempty"`
	Array  []string          `json:"array,omitempty"`
}

func main() {
	_ = examplev1.AddToScheme(scheme.Scheme)
	ctx := plugin.MustInitWithOptions(plugin.Options{
		RegisterMappings: []resources.BuildMapper{
			resources.CreateSecretsMapper,
		},
	})
	err := validateConfig()
	if err != nil {
		klog.Fatalf("validate config: %v", err)
	}
	plugin.MustRegister(syncers.NewServiceHook())
	plugin.MustRegister(syncers.NewPodHook())
	plugin.MustRegister(syncers.NewSecretHook())
	plugin.MustRegister(syncers.NewMyDeploymentSyncer(ctx))
	carSyncer, err := syncers.NewCarSyncer(ctx)
	if err != nil {
		klog.Fatalf("new car syncer: %v", err)
	}
	plugin.MustRegister(carSyncer)
	plugin.MustRegister(syncers.DummyInterceptor{})

	klog.Info("finished registering the plugins")
	plugin.MustStart()
}

func validateConfig() error {
	// verify config
	pConfig := &PluginConfig{}
	klog.Info(os.Getenv("PLUGIN_CONFIG"))
	err := plugin.UnmarshalConfig(pConfig)
	if err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	if pConfig.Int != 123 {
		return fmt.Errorf("expected int to be 123, got %v", pConfig.Int)
	}
	if pConfig.String != "string" {
		return fmt.Errorf("expected string to be string, got %v", pConfig.String)
	}
	if len(pConfig.Map) != 1 || pConfig.Map["entry"] != "entry" {
		return fmt.Errorf("expected map to contain entry, got %v", pConfig.Map)
	}
	if len(pConfig.Array) != 1 || pConfig.Array[0] != "entry" {
		return fmt.Errorf("expected map to contain entry, got %v", pConfig.Map)
	}

	return nil
}
