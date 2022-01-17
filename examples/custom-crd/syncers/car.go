package syncers

import (
	examplev1 "github.com/loft-sh/vcluster-example/apis/v1"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster-sdk/syncer"
	synccontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/syncer/translator"
	"k8s.io/apimachinery/pkg/api/equality"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Make sure our scheme is registered
	_ = examplev1.AddToScheme(plugin.Scheme)
}

func NewCarSyncer(ctx *synccontext.RegisterContext) syncer.Object {
	return &carSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "car", &examplev1.Car{}),
	}
}

type carSyncer struct {
	translator.NamespacedTranslator
}

func (s *carSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownCreate(ctx, vObj, s.TranslateMetadata(vObj).(*examplev1.Car))
}

func (s *carSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	return s.SyncDownUpdate(ctx, vObj, s.translateUpdate(pObj.(*examplev1.Car), vObj.(*examplev1.Car)))
}

func (s *carSyncer) translateUpdate(pObj, vObj *examplev1.Car) *examplev1.Car {
	var updated *examplev1.Car

	// check annotations & labels
	changed, updatedAnnotations, updatedLabels := s.TranslateMetadataUpdate(vObj, pObj)
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
