package syncers

import (
	"fmt"

	"github.com/loft-sh/vcluster-sdk/syncer"
	synccontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// this particular label key is just an example
	// it does't have any additional meaning to vcluster
	ManagedByPluginLabelKey   = "plugin.vcluster.loft.sh/managed-by"
	ManagedByPluginLabelValue = "pull-secret-sync"
	ManagedByVclusterLabelKey = "vcluster.loft.sh/managed-by"
)

func NewPullSecretSyncer(ctx *synccontext.RegisterContext, labelSelector metav1.LabelSelector, destinationNamespace string) syncer.Syncer {
	return &pullSecretSyncer{
		targetNamespace:      ctx.TargetNamespace,
		LabelSelector:        labelSelector,
		DestinationNamespace: destinationNamespace,
	}
}

type pullSecretSyncer struct {
	targetNamespace      string
	LabelSelector        metav1.LabelSelector
	DestinationNamespace string
}

func (s *pullSecretSyncer) Name() string {
	return "pull-secret-syncer"
}

func (s *pullSecretSyncer) Resource() client.Object {
	return &corev1.Secret{}
}

var _ syncer.Starter = &pullSecretSyncer{}

// ReconcileStart is executed before the syncer or fake syncer reconcile starts and can return
// true if the rest of the reconcile should be skipped. If an error is returned, the reconcile
// will fail and try to requeue.
func (s *pullSecretSyncer) ReconcileStart(ctx *synccontext.SyncContext, req ctrl.Request) (bool, error) {
	// reconcile can be skipped if the Secret that triggered this reconciliation request
	// is not from the DestinationNamespace
	return req.Namespace != s.DestinationNamespace, nil
}

func (s *pullSecretSyncer) ReconcileEnd() {
	//NOOP
}

var _ syncer.UpSyncer = &pullSecretSyncer{}

func (s *pullSecretSyncer) SyncUp(ctx *synccontext.SyncContext, pObj client.Object) (ctrl.Result, error) {
	pSecret := pObj.(*corev1.Secret)
	if pSecret.Type != corev1.SecretTypeDockerConfigJson {
		// ignore secrets that are not of "pull secret" type
		return ctrl.Result{}, nil
	}
	if pSecret.GetLabels()[ManagedByVclusterLabelKey] != "" {
		// ignore Secrets synced to the host by the vcluster
		return ctrl.Result{}, nil
	}

	labels := map[string]string{
		ManagedByPluginLabelKey: ManagedByPluginLabelValue,
	}
	for k, v := range pSecret.GetLabels() {
		labels[k] = v
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   s.DestinationNamespace,
			Name:        pObj.GetName(),
			Annotations: pObj.GetAnnotations(),
			Labels:      labels,
		},
		Immutable: pSecret.Immutable,
		Data:      pSecret.Data,
		Type:      corev1.SecretTypeDockerConfigJson,
	}
	err := ctx.VirtualClient.Create(ctx.Context, secret)
	if err == nil {
		ctx.Log.Infof("created pull secret %s/%s", secret.GetNamespace(), secret.GetName())
	} else {
		err = fmt.Errorf("failed to create pull secret %s/%s: %v", secret.GetNamespace(), secret.GetName(), err)
	}
	return ctrl.Result{}, err
}

func (s *pullSecretSyncer) Sync(ctx *synccontext.SyncContext, pObj client.Object, vObj client.Object) (ctrl.Result, error) {
	pSecret := pObj.(*corev1.Secret)
	if pSecret.Type != corev1.SecretTypeDockerConfigJson {
		if vObj.GetLabels()[ManagedByPluginLabelKey] == ManagedByPluginLabelValue {
			// delete synced secret if the type of a the host secret is no longer a pull secret
			err := ctx.VirtualClient.Delete(ctx.Context, vObj)
			if err == nil {
				ctx.Log.Infof("deleted pull secret %s/%s because host secret is no longer equal to %s", vObj.GetNamespace(), vObj.GetName(), corev1.SecretTypeDockerConfigJson)
			} else {
				err = fmt.Errorf("failed to delete pull secret %s/%s: %v", vObj.GetNamespace(), vObj.GetName(), err)
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	vSecret := vObj.(*corev1.Secret)
	updated := s.translateUpdateUp(pSecret, vSecret)
	if updated == nil {
		// no update is needed
		return ctrl.Result{}, nil
	}

	err := ctx.VirtualClient.Update(ctx.Context, updated)
	if err == nil {
		ctx.Log.Infof("update pull secret %s/%s", vObj.GetNamespace(), vObj.GetName())
	} else {
		err = fmt.Errorf("failed to update pull secret %s/%s: %v", vObj.GetNamespace(), vObj.GetName(), err)
	}

	return ctrl.Result{}, err
}

func (s *pullSecretSyncer) SyncDown(ctx *synccontext.SyncContext, vObj client.Object) (ctrl.Result, error) {
	// this is called when the secret in the host gets removed
	// or if the vObj is an unrelated Secret created in vcluster

	// check if this particular secret was created by this plugin
	if vObj.GetLabels()[ManagedByPluginLabelKey] == ManagedByPluginLabelValue {
		// delete synced secret because the host secret was deleted
		err := ctx.VirtualClient.Delete(ctx.Context, vObj)
		if err == nil {
			ctx.Log.Infof("deleted pull secret %s/%s because host secret no longer exists", vObj.GetNamespace(), vObj.GetName())
		} else {
			err = fmt.Errorf("failed to delete pull secret %s/%s: %v", vObj.GetNamespace(), vObj.GetName(), err)
		}
		return ctrl.Result{}, err
	}
	// ignore all unrelated Secrets
	return ctrl.Result{}, nil
}

// IsManaged determines if a physical object is managed by the vcluster
func (s *pullSecretSyncer) IsManaged(pObj client.Object) (bool, error) {
	// we will consider all Secrets as managed in order to reconcile
	// when a secret type changes, and we will check the type
	// in the Sync and SyncUp methods and ignore the irrelevant ones
	return true, nil
}

// VirtualToPhysical translates a virtual name to a physical name
func (s *pullSecretSyncer) VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName {
	// the secret that is being mirrored by a particular vObj secret
	// is located in the "TargetNamespace" of the host cluster
	return types.NamespacedName{
		Namespace: s.targetNamespace,
		Name:      req.Name,
	}
}

// PhysicalToVirtual translates a physical name to a virtual name
func (s *pullSecretSyncer) PhysicalToVirtual(pObj client.Object) types.NamespacedName {
	// the secret mirrored to vcluster is always named the same as the
	// original in the host, and it is located in the DestinationNamespace
	return types.NamespacedName{
		Namespace: s.DestinationNamespace,
		Name:      pObj.GetName(),
	}
}

func (s *pullSecretSyncer) translateUpdateUp(pObj, vObj *corev1.Secret) *corev1.Secret {
	var updated *corev1.Secret

	// sync annotations
	// we sync all of them from the host and remove any added in the vcluster
	if !equality.Semantic.DeepEqual(vObj.GetAnnotations(), pObj.GetAnnotations()) {
		updated = newIfNil(updated, vObj)
		updated.Annotations = pObj.GetAnnotations()
	}

	// check labels
	// we sync all of them from the host, add one more to be able to detect
	// secrets synced by this plugin, and we remove any added in the vcluster
	expectedLabels := map[string]string{
		ManagedByPluginLabelKey: ManagedByPluginLabelValue,
	}
	for k, v := range pObj.GetLabels() {
		expectedLabels[k] = v
	}
	if !equality.Semantic.DeepEqual(vObj.GetLabels(), expectedLabels) {
		updated = newIfNil(updated, vObj)
		updated.Labels = expectedLabels
	}

	//TODO: update .Immutable field? Will need to recreate the Secret if it was immutable and now suddenly Immutable!=true
	// this could happen if vcluster was offline while host secret was recreated in the meantime

	// check data
	if !equality.Semantic.DeepEqual(vObj.Data, pObj.Data) {
		updated = newIfNil(updated, vObj)
		updated.Data = pObj.Data
	}

	return updated
}

func newIfNil(updated *corev1.Secret, pObj *corev1.Secret) *corev1.Secret {
	if updated == nil {
		return pObj.DeepCopy()
	}
	return updated
}
