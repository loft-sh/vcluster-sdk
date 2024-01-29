package syncers

import (
	"context"
	"os"

	examplev1 "github.com/loft-sh/vcluster-sdk/e2e/test_plugin/apis/v1"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer/translator"
	synctypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCarSyncer(ctx *synccontext.RegisterContext) synctypes.Base {
	return &carSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "car", &examplev1.Car{}),
	}
}

type carSyncer struct {
	translator.NamespacedTranslator
}

var _ synctypes.Initializer = &carSyncer{}

func (s *carSyncer) Init(ctx *synccontext.RegisterContext) error {
	out, err := os.ReadFile("manifests/crds.yaml")
	if err != nil {
		return err
	}

	gvk := examplev1.GroupVersion.WithKind("Car")
	err = util.EnsureCRD(ctx.Context, ctx.PhysicalManager.GetConfig(), out, gvk)
	if err != nil {
		return err
	}

	_, _, err = translate.EnsureCRDFromPhysicalCluster(ctx.Context, ctx.PhysicalManager.GetConfig(), ctx.VirtualManager.GetConfig(), gvk)
	return err
}

func (s *carSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.TranslateMetadata(ctx.Context, vObj).(*examplev1.Car))
}

func (s *carSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownUpdate(ctx, vObj, s.translateUpdate(ctx.Context, pObj.(*examplev1.Car), vObj.(*examplev1.Car)))
}

func (s *carSyncer) translateUpdate(ctx context.Context, pObj, vObj *examplev1.Car) *examplev1.Car {
	var updated *examplev1.Car

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(ctx, vObj, pObj)
	if changed {
		updated = newIfNil(updated, pObj)
		updated.Labels = updatedLabels
		updated.Annotations = updatedAnnotations
	}

	// check spec
	if !equality.Semantic.DeepEqual(vObj.Spec, pObj.Spec) {
		updated = newIfNil(updated, pObj)
		updated.Spec = vObj.Spec
	}

	return updated
}

func newIfNil(updated *examplev1.Car, pObj *examplev1.Car) *examplev1.Car {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
