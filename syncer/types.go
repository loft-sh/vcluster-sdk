package syncer

import (
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Base is a basic entity, which identifies the action through its name. Additional
// functionality can be added to this basic identity through implementing other interfaces
// below. The functionality is additive, which means that actions that implement multiple
// interfaces are valid. The syncer, fake syncer or hook interface are not needed to get implemented
// and actions that implement for example only the Initializer or IndicesRegisterer interfaces
// are valid.
type Base interface {
	Name() string
}

// Syncer is a controller that runs in the virtual cluster and is fed events based on
// the base object that is defined through the Base interface. vcluster will automatically
// watch on events in the host cluster and feds host cluster events into the virtual cluster
// controller by converting the involved object names based on the NameTranslator interface
// implementation. Various default implementations can be found in the translator package.
type Syncer interface {
	Base
	translator.NameTranslator

	// Resource is the typed resource (e.g. &corev1.Pod{}) that should get synced between
	// host and virtual cluster. Namespaced as well as cluster scoped resources are possible
	// here.
	Resource() client.Object

	// SyncDown defines the action that should be taken by the syncer if a virtual cluster object
	// exists, but has no corresponding physical cluster object yet. Typically, the physical cluster
	// object would get synced down from the virtual cluster to the host cluster in this scenario.
	SyncDown(ctx *context.SyncContext, vObj client.Object) (ctrl.Result, error)

	// Sync defines the action that should be taken by the syncer if a virtual cluster object and
	// physical cluster object exist and either one of them has changed. The syncer is expected
	// to reconcile in this case without knowledge of which object has actually changed. This
	// is needed to avoid race conditions and defining a clear hierarchy what fields should be managed
	// by which cluster. For example, for pods you would want to sync down (virtual -> physical)
	// spec changes, while you would want to sync up (physical -> virtual) status changes, as those
	// would get set only by the physical host cluster.
	Sync(ctx *context.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error)
}

// UpSyncer adds functionality in the case a syncer needs to control the situation where
// only a physical object is present that should get somehow modified or synced into the virtual cluster.
// If this interface is not implemented by the syncer, vcluster will automatically delete the physical
// object in this case instead, as no corresponding virtual object does exist anymore.
type UpSyncer interface {
	// SyncUp defines the action that should be taken by the syncer if a physical cluster object
	// exists, but not corresponding virtual object is found. Typically, you want to delete this
	// object, which is the default implementation if this interface is not implemented. However,
	// there are situations where this shouldn't be the case, for example if objects should get
	// synced from the host cluster to the virtual cluster (like nodes), in this case the syncer
	// can implement this interface to gain full control over this situation.
	SyncUp(ctx *context.SyncContext, pObj client.Object) (ctrl.Result, error)
}

// FakeSyncer is a controller that runs in the virtual cluster and creates fake objects based
// on a change in a related object, that needs another object to function properly, but vcluster
// does not have access to sync this object or the object itself is not relevant. For example,
// fake syncers are used to mock nodes that are bound by pods without vcluster needing permissions
// to actually sync to those nodes. Usually a fake syncer requires events from other objects,
// so implementing the ControllerModifier interface is very common.
type FakeSyncer interface {
	Base

	// Resource is the typed resource (e.g. &corev1.Node{}) that should get created in the
	// virtual cluster. Namespaced as well as cluster scoped resources are possible
	// here.
	Resource() client.Object

	// FakeSyncUp defines the action that should get taken if the given resource name is
	// expected to exist in the virtual cluster, but does not yet exist. For example,
	// for faking nodes in the pod syncer, this would be the spec.nodeName of a pod that
	// will be fed into the fake syncer, which is then expected to create a fake node for
	// this.
	FakeSyncUp(ctx *context.SyncContext, req types.NamespacedName) (ctrl.Result, error)

	// FakeSync defines the action that should get taken if the given resource name
	// already exists in the virtual cluster, but the resource has somehow changed or
	// a related resource was changed.
	FakeSync(ctx *context.SyncContext, vObj client.Object) (ctrl.Result, error)
}

// Starter allows a syncer or fake syncer to gain more control over the reconcile process
// by running an action before every reconcile and after every reconcile. This action can
// also stop the reconcile from proceeding which essentially means that through this interface
// you can bypass the default syncer logic enterily.
type Starter interface {
	// ReconcileStart is executed before the syncer or fake syncer reconcile starts and can return
	// true if the rest of the reconcile should be skipped. If an error is returned, the reconcile
	// will fail and try to requeue.
	ReconcileStart(ctx *context.SyncContext, req ctrl.Request) (bool, error)

	// ReconcileEnd is executed after a reconcile was running through, no matter if the reconcile
	// has failed or not.
	ReconcileEnd()
}

// Initializer is used to initialize a certain syncer. This can be used for example to sync certain
// crds or apply manifests you'll need in the virtual cluster. This interface does not require
// the syncer or fake syncer interface to be implemented. The order in which vcluster will execute
// those before functions is:
// 1. Init()
// 2. RegisterIndices()
// 3. ModifyController() (Only if Syncer or FakeSyncer is implemented)
//
// Keep in mind that the context's managers are not started at this point, which means that accessing
// the client's will not work and new clients are needed to be generated, if kubernetes access is needed:
// kubeClient, err := kubernetes.NewClientFor(ctx.VirtualManager.GetConfig())
type Initializer interface {
	Init(registerContext *context.RegisterContext) error
}

// IndicesRegisterer registers additional indices for the syncers.
type IndicesRegisterer interface {
	RegisterIndices(ctx *context.RegisterContext) error
}

// ControllerModifier is used to modify the created controller for the syncer by watching on other
// relevant resources or changing the controller's options. The cache's will be synced at this point,
// and it is safe to access the ctx.VirtualManager.GetClient() or ctx.PhysicalManager.GetClient().
type ControllerModifier interface {
	ModifyController(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error)
}

// ControllerStarter is a generic controller that can be used if the syncer abstraction does not fit
// the use case
type ControllerStarter interface {
	Register(ctx *context.RegisterContext) error
}
