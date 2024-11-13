package runtimeclasses

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/pro"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	nodev1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(_ *synccontext.RegisterContext) (syncertypes.Object, error) {
	mapper, err := generic.NewMirrorMapper(&nodev1.RuntimeClass{})
	if err != nil {
		return nil, err
	}

	return &runtimeClassSyncer{
		Mapper: mapper,
	}, nil
}

type runtimeClassSyncer struct {
	synccontext.Mapper
}

func (i *runtimeClassSyncer) Name() string {
	return "runtimeclass"
}

func (i *runtimeClassSyncer) Resource() client.Object {
	return &nodev1.RuntimeClass{}
}

var _ syncertypes.Syncer = &runtimeClassSyncer{}

func (i *runtimeClassSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer(i)
}

func (i *runtimeClassSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*nodev1.RuntimeClass]) (ctrl.Result, error) {
	vObj := translate.CopyObjectWithName(event.Host, types.NamespacedName{Name: event.Host.Name, Namespace: event.Host.Namespace}, false)

	// Apply pro patches
	err := pro.ApplyPatchesVirtualObject(ctx, nil, vObj, event.Host, ctx.Config.Sync.FromHost.RuntimeClasses.Patches, true)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error applying patches: %w", err)
	}

	ctx.Log.Infof("create runtime class %s, because it does not exist in virtual cluster", vObj.Name)
	return ctrl.Result{}, ctx.VirtualClient.Create(ctx, vObj)
}

func (i *runtimeClassSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*nodev1.RuntimeClass]) (_ ctrl.Result, retErr error) {
	patch, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual, patcher.TranslatePatches(ctx.Config.Sync.FromHost.RuntimeClasses.Patches, true))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}
	defer func() {
		if err := patch.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = utilerrors.NewAggregate([]error{retErr, err})
		}
	}()

	event.Virtual.Annotations = translate.VirtualAnnotations(event.Host, event.Virtual)
	event.Virtual.Labels = translate.VirtualLabels(event.Host, event.Virtual)
	event.Virtual.Handler = event.Host.Handler
	event.Virtual.Overhead = event.Host.Overhead
	event.Virtual.Scheduling = event.Host.Scheduling
	return ctrl.Result{}, nil
}

func (i *runtimeClassSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*nodev1.RuntimeClass]) (ctrl.Result, error) {
	ctx.Log.Infof("delete virtual runtime class %s, because physical object is missing", event.Virtual.Name)
	return ctrl.Result{}, ctx.VirtualClient.Delete(ctx, event.Virtual)
}
