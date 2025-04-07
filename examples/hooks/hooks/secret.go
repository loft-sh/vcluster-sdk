package hooks

import (
	"context"
	"fmt"

	"github.com/loft-sh/vcluster-sdk/plugin"
	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	synctypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewSecretHook() plugin.ClientHook {
	return &secretHook{}
}

// Purpose of this hook is to create a secret in the virtual cluster that is always synced
// to the host cluster, without directly setting the annotation on the secret.
type secretHook struct{}

var _ synctypes.ControllerStarter = &secretHook{}

func (s *secretHook) Register(ctx *synccontext.RegisterContext) error {
	virtualClient, err := client.New(ctx.VirtualManager.GetConfig(), client.Options{
		Scheme: ctx.VirtualManager.GetScheme(),
		Mapper: ctx.VirtualManager.GetRESTMapper(),
	})
	if err != nil {
		return err
	}

	mySecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"my-data": []byte("my-data"),
		},
	}
	err = virtualClient.Create(ctx.Context, mySecret)
	if err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("create secret: %w", err)
	}

	return nil
}

func (s *secretHook) Name() string {
	return "secret-hook"
}

func (s *secretHook) Resource() client.Object {
	return &corev1.Secret{}
}

var _ plugin.MutateGetVirtual = &serviceHook{}

func (s *secretHook) MutateGetVirtual(ctx context.Context, obj client.Object) (client.Object, error) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil, fmt.Errorf("object %v is not a secret", obj)
	} else if secret.Name != "test" || secret.Namespace != "default" {
		return secret, nil
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}

	// Force sync the secret to the host cluster
	secret.Annotations["vcluster.loft.sh/force-sync"] = "true"
	return secret, nil
}
