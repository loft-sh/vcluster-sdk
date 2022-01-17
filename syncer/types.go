package syncer

import (
	"github.com/loft-sh/vcluster-sdk/log"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Object is the base for syncers and returns the typed object that should get acted on.
type Object interface {
	New() client.Object
}

// Syncer is the main implementation, which handles syncing objects between the host cluster
// and virtual cluster.
type Syncer interface {
	Object
	Translator

	Forward(ctx Context, vObj client.Object, log log.Logger) (ctrl.Result, error)
	Update(ctx Context, pObj client.Object, vObj client.Object, log log.Logger) (ctrl.Result, error)
}

type BackwardSyncer interface {
	Backward(ctx Context, pObj client.Object, log log.Logger) (ctrl.Result, error)
}

type FakeSyncer interface {
	Object

	Create(ctx Context, req types.NamespacedName, log log.Logger) (ctrl.Result, error)
	Update(ctx Context, vObj client.Object, log log.Logger) (ctrl.Result, error)
}

type Starter interface {
	ReconcileStart(ctx Context, req ctrl.Request) (bool, error)
	ReconcileEnd()
}
