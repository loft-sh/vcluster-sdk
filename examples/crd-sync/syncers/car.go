package syncers

import (
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	examplev1 "github.com/loft-sh/vcluster-example/apis/v1"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	synctypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Make sure our scheme is registered
	_ = examplev1.AddToScheme(scheme.Scheme)
}

func NewCarSyncer(ctx *synccontext.RegisterContext) synctypes.Base {
	mapper, err := ctx.Mappings.ByGVK(schema.GroupVersionKind{
		Group:   examplev1.SchemeGroupVersion.Group,
		Version: examplev1.SchemeGroupVersion.Version,
		Kind:    "Car",
	})
	if err != nil {
		klog.FromContext(ctx).Error(err, "unable to get mapping for examplev1.Car")
		return nil
	}
	s := &carSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "car", &examplev1.Car{}, mapper),
	}
	return s
}

type carSyncer struct {
	synctypes.GenericTranslator
}

func (s *carSyncer) Syncer() synctypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*examplev1.Car](s)
}

var _ synctypes.ControllerStarter = &carSyncer{}

func (s *carSyncer) Register(ctx *synccontext.RegisterContext) error {
	_, _, err := translate.EnsureCRDFromPhysicalCluster(ctx.Context, ctx.HostManager.GetConfig(), ctx.VirtualManager.GetConfig(), examplev1.GroupVersion.WithKind("Car"))
	return err
}

var _ synctypes.Syncer = &carSyncer{}

func (s *carSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*examplev1.Car]) (ctrl.Result, error) {
	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.GetName(), Namespace: event.Virtual.GetNamespace()}, event.Virtual))
	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *carSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*examplev1.Car]) (ctrl.Result, error) {
	return patcher.CreateHostObject(ctx, event.Virtual, event.Host, s.EventRecorder(), true)
}

func (s *carSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*examplev1.Car]) (ctrl.Result, error) {
	if event.VirtualOld != nil || translate.ShouldDeleteHostObject(event.Host) {
		// virtual object is not here anymore, so we delete
		return patcher.DeleteHostObject(ctx, event.Host, event.VirtualOld, "virtual object was deleted")
	}
	return ctrl.Result{}, nil
}
