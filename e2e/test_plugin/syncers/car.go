package syncers

import (
	_ "embed"
	"fmt"

	examplev1 "github.com/loft-sh/vcluster-sdk/e2e/test_plugin/apis/v1"
	"github.com/loft-sh/vcluster/pkg/mappings/generic"
	"github.com/loft-sh/vcluster/pkg/patcher"
	"github.com/loft-sh/vcluster/pkg/syncer"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	"github.com/loft-sh/vcluster/pkg/syncer/translator"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed car.crd.yaml
var carCRD string

func NewCarSyncer(ctx *synccontext.RegisterContext) (syncertypes.Base, error) {
	err := util.EnsureCRD(ctx.Context, ctx.HostManager.GetConfig(), []byte(carCRD), examplev1.SchemeGroupVersion.WithKind("Car"))
	if err != nil {
		return nil, err
	}

	err = util.EnsureCRD(ctx.Context, ctx.VirtualManager.GetConfig(), []byte(carCRD), examplev1.SchemeGroupVersion.WithKind("Car"))
	if err != nil {
		return nil, err
	}

	mapper, err := generic.NewMapper(ctx, &examplev1.Car{}, translate.Default.HostName)
	if err != nil {
		return nil, err
	}

	return &carSyncer{
		GenericTranslator: translator.NewGenericTranslator(ctx, "car", &examplev1.Car{}, mapper),
	}, nil
}

type carSyncer struct {
	syncertypes.GenericTranslator
}

var _ syncertypes.Syncer = &carSyncer{}

func (s *carSyncer) Syncer() syncertypes.Sync[client.Object] {
	return syncer.ToGenericSyncer[*examplev1.Car](s)
}

func (s *carSyncer) SyncToHost(ctx *synccontext.SyncContext, event *synccontext.SyncToHostEvent[*examplev1.Car]) (ctrl.Result, error) {
	pObj := translate.HostMetadata(event.Virtual, s.VirtualToHost(ctx, types.NamespacedName{Name: event.Virtual.Name, Namespace: event.Virtual.Namespace}, event.Virtual))
	return patcher.CreateHostObject(ctx, event.Virtual, pObj, s.EventRecorder(), true)
}

func (s *carSyncer) Sync(ctx *synccontext.SyncContext, event *synccontext.SyncEvent[*examplev1.Car]) (_ ctrl.Result, retErr error) {
	patchHelper, err := patcher.NewSyncerPatcher(ctx, event.Host, event.Virtual)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("new syncer patcher: %w", err)
	}

	defer func() {
		if err := patchHelper.Patch(ctx, event.Host, event.Virtual); err != nil {
			retErr = errors.NewAggregate([]error{retErr, err})
		}
		if retErr != nil {
			s.EventRecorder().Eventf(event.Virtual, "Warning", "SyncError", "Error syncing: %v", retErr)
		}
	}()

	// any changes made below here are correctly synced

	// sync metadata
	event.Host.Annotations = translate.HostAnnotations(event.Virtual, event.Host)
	event.Host.Labels = translate.HostLabels(event.Virtual, event.Host)

	// sync virtual to host
	event.Host.Spec = event.Virtual.Spec

	return ctrl.Result{}, nil
}

func (s *carSyncer) SyncToVirtual(ctx *synccontext.SyncContext, event *synccontext.SyncToVirtualEvent[*examplev1.Car]) (ctrl.Result, error) {
	// virtual object is not here anymore, so we delete
	return patcher.DeleteHostObject(ctx, event.Host, nil, "virtual object was deleted")
}
