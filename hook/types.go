package hook

import (
	"context"
	"github.com/loft-sh/vcluster-sdk/syncer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClientHook interface {
	syncer.Base

	// Resource is the typed resource (e.g. &corev1.Pod{}) that should get mutated.
	Resource() client.Object
}

type MutateCreateVirtual interface {
	MutateCreateVirtual(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateUpdateVirtual interface {
	MutateUpdateVirtual(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateDeleteVirtual interface {
	MutateDeleteVirtual(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateGetVirtual interface {
	MutateGetVirtual(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateCreatePhysical interface {
	MutateCreatePhysical(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateUpdatePhysical interface {
	MutateUpdatePhysical(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateDeletePhysical interface {
	MutateDeletePhysical(ctx context.Context, obj client.Object) (client.Object, error)
}

type MutateGetPhysical interface {
	MutateGetPhysical(ctx context.Context, obj client.Object) (client.Object, error)
}
