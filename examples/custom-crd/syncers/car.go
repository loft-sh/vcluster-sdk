package syncers

import (
	"fmt"
	"github.com/FabianKramm/vcluster/pkg/constants"
	examplev1 "github.com/loft-sh/vcluster-example/apis/v1"
	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	// Make sure our scheme is registered
	_ = examplev1.AddToScheme(plugin.Scheme)
}

func NewCarSyncer(ctx *context.SyncContext) (syncer.Object, error) {
	return &configMapSyncer{
		NamespacedTranslator: translator.NewNamespacedTranslator(ctx, "configmap", &corev1.ConfigMap{}),
	}, nil
}

type carSyncer struct {
	translator.NamespacedTranslator
}

func (s *configMapSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	createNeeded, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !createNeeded {
		return ctrl.Result{}, nil
	}

	return s.SyncDownCreate(ctx, vObj, s.translate(vObj.(*corev1.ConfigMap)))
}

func (s *configMapSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	used, err := s.isConfigMapUsed(ctx, vObj)
	if err != nil {
		return ctrl.Result{}, err
	} else if !used {
		pConfigMap, _ := meta.Accessor(pObj)
		ctx.Log.Infof("delete physical config map %s/%s, because it is not used anymore", pConfigMap.GetNamespace(), pConfigMap.GetName())
		err = ctx.PhysicalClient.Delete(ctx.Context, pObj)
		if err != nil {
			ctx.Log.Infof("error deleting physical object %s/%s in physical cluster: %v", pConfigMap.GetNamespace(), pConfigMap.GetName(), err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// did the configmap change?
	return s.SyncDownUpdate(ctx, vObj, s.translateUpdate(pObj.(*corev1.ConfigMap), vObj.(*corev1.ConfigMap)))
}

func (s *configMapSyncer) isConfigMapUsed(ctx *synccontext.SyncContext, vObj runtime.Object) (bool, error) {
	configMap, ok := vObj.(*corev1.ConfigMap)
	if !ok || configMap == nil {
		return false, fmt.Errorf("%#v is not a config map", vObj)
	}

	podList := &corev1.PodList{}
	err := ctx.VirtualClient.List(context.TODO(), podList, client.MatchingFields{constants.IndexByConfigMap: configMap.Namespace + "/" + configMap.Name})
	if err != nil {
		return false, err
	}

	return len(podList.Items) > 0, nil
}
