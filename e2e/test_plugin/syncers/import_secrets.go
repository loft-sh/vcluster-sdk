package syncers

import (
	"context"
	"fmt"
	"strings"

	"github.com/loft-sh/vcluster/pkg/constants"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	vImportedAnnotation = "vcluster.loft.sh/imported"
	pImportAnnotation   = "vcluster.loft.sh/import"
)

func NewImportSecrets(ctx *synccontext.RegisterContext) (syncertypes.Object, error) {
	// get secret mapper
	secretsMapper, err := ctx.Mappings.ByGVK(corev1.SchemeGroupVersion.WithKind("Secret"))
	if err != nil {
		return nil, err
	}

	return &importSecretSyncer{
		secretsMapper: secretsMapper,

		syncContext: ctx.ToSyncContext("import-secret-syncer"),

		pClient: ctx.PhysicalManager.GetClient(),
		vClient: ctx.VirtualManager.GetClient(),
	}, nil
}

type importSecretSyncer struct {
	secretsMapper synccontext.Mapper

	syncContext *synccontext.SyncContext

	pClient client.Client
	vClient client.Client
}

func (s *importSecretSyncer) Name() string {
	return "import-secret-syncer"
}

func (s *importSecretSyncer) Resource() client.Object {
	return &corev1.Secret{}
}

var _ syncertypes.Syncer = &carSyncer{}

func (s *importSecretSyncer) Register(ctx *synccontext.RegisterContext) error {
	return ctrl.NewControllerManagedBy(ctx.PhysicalManager).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
			CacheSyncTimeout:        constants.DefaultCacheSyncTimeout,
		}).
		Named(s.Name()).
		For(s.Resource()).
		Complete(s)
}

func (s *importSecretSyncer) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	// get secret
	pSecret := &corev1.Secret{}
	err := s.pClient.Get(ctx, req.NamespacedName, pSecret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, s.deleteVirtualSecret(ctx, req.NamespacedName)
		}

		return ctrl.Result{}, err
	}

	// ignore Secrets synced to the host by the vcluster
	if isManaged, err := s.secretsMapper.IsManaged(s.syncContext, pSecret); isManaged || err != nil {
		if err != nil {
			klog.FromContext(ctx).Error(err, "is managed secret")
		}

		return ctrl.Result{}, nil
	} else if pSecret.Labels != nil && pSecret.Labels[translate.MarkerLabel] != "" {
		return ctrl.Result{}, nil
	}

	// try to parse import annotation
	namespaceName := parseFromAnnotation(pSecret.Annotations, pImportAnnotation)
	if namespaceName.Name == "" {
		return ctrl.Result{}, s.deleteVirtualSecret(ctx, req.NamespacedName)
	}

	// check if namespace is there
	err = s.vClient.Get(ctx, types.NamespacedName{Name: namespaceName.Namespace}, &corev1.Namespace{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// create namespace
		s.syncContext.Log.Infof("create namespace %s", namespaceName.Namespace)
		err = s.vClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName.Namespace}})
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("creating namespace %s: %w", namespaceName.Namespace, err)
		}
	}

	// check if virtual secret exists
	vSecret := &corev1.Secret{}
	err = s.vClient.Get(ctx, namespaceName, vSecret)
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("get virtual secret: %w", err)
		}

		vSecret = nil
	}

	// check if create or update
	if vSecret == nil {
		// create virtual secret
		vSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespaceName.Namespace,
				Name:        namespaceName.Name,
				Annotations: getVirtualAnnotations(pSecret),
				Labels:      pSecret.Labels,
			},
			Immutable: pSecret.Immutable,
			Data:      pSecret.Data,
			Type:      pSecret.Type,
		}
		s.syncContext.Log.Infof("import secret %s/%s into %s/%s", pSecret.GetNamespace(), pSecret.GetName(), vSecret.Namespace, vSecret.Name)
		err = s.vClient.Create(ctx, vSecret)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to import secret %s/%s: %v", pSecret.GetNamespace(), pSecret.GetName(), err)
		}

		return ctrl.Result{}, nil
	}

	// check if update is needed
	updated := s.translateUpdateUp(pSecret, vSecret)
	if updated == nil {
		// no update is needed
		return ctrl.Result{}, nil
	}

	// update secret
	s.syncContext.Log.Infof("update imported secret %s/%s", vSecret.GetNamespace(), vSecret.GetName())
	err = s.vClient.Update(ctx, updated)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update imported secret %s/%s: %v", vSecret.GetNamespace(), vSecret.GetName(), err)
	}

	return ctrl.Result{}, nil
}

func (s *importSecretSyncer) translateUpdateUp(pObj, vObj *corev1.Secret) *corev1.Secret {
	var updated *corev1.Secret

	// check annotations
	expectedAnnotations := getVirtualAnnotations(pObj)
	if !equality.Semantic.DeepEqual(vObj.GetAnnotations(), expectedAnnotations) {
		updated = vObj.DeepCopy()
		updated.Annotations = expectedAnnotations
	}

	// check labels
	if !equality.Semantic.DeepEqual(vObj.GetLabels(), pObj.GetLabels()) {
		if updated == nil {
			updated = vObj.DeepCopy()
		}
		updated.Labels = pObj.GetLabels()
	}

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		if updated == nil {
			updated = vObj.DeepCopy()
		}
		updated.Data = pObj.Data
	}

	return updated
}

func (s *importSecretSyncer) deleteVirtualSecret(ctx context.Context, req types.NamespacedName) error {
	vSecrets := &corev1.SecretList{}
	err := s.vClient.List(ctx, vSecrets)
	if err != nil {
		return fmt.Errorf("list secrets: %w", err)
	}

	for _, vSecret := range vSecrets.Items {
		if vSecret.Annotations[vImportedAnnotation] == req.String() {
			klog.FromContext(ctx).Info("Delete virtual secret because it was imported and host secret got deleted")
			err = s.vClient.Delete(ctx, &vSecret)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getVirtualAnnotations(pSecret *corev1.Secret) map[string]string {
	annotations := map[string]string{}
	for k, v := range pSecret.Annotations {
		if k == pImportAnnotation {
			continue
		}

		annotations[k] = v
	}

	annotations[vImportedAnnotation] = pSecret.Namespace + "/" + pSecret.Name
	return annotations
}

func parseFromAnnotation(annotations map[string]string, annotation string) types.NamespacedName {
	if annotations == nil || annotations[annotation] == "" {
		return types.NamespacedName{}
	}

	splitted := strings.Split(annotations[annotation], "/")
	if len(splitted) != 2 {
		klog.Infof("Retrieved malformed import annotation %s: %s, expected NAMESPACE/NAME", annotation, annotations[annotation])

		return types.NamespacedName{}
	}

	return types.NamespacedName{
		Namespace: splitted[0],
		Name:      splitted[1],
	}
}
