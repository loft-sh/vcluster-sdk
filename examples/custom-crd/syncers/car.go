package syncers

import (
	examplev1 "github.com/loft-sh/vcluster-example/apis/v1"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster-sdk/syncer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Make sure our scheme is registered
	_ = examplev1.AddToScheme(plugin.Scheme)
}

type CarSyncer struct{}

func (c *CarSyncer) New() client.Object {
	return &examplev1.Car{}
}

func (c *CarSyncer) Translate(ctx syncer.Context, vObj client.Object) (pObj client.Object, err error) {
	return nil, nil
}

func (c *CarSyncer) TranslateUpdate(ctx syncer.Context, pObj, vObj client.Object) (pObjUpdated client.Object, err error) {
	return nil, nil
}
