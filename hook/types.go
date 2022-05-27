package hook

import (
	"context"
	"github.com/loft-sh/vcluster-sdk/syncer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClientHook tells the sdk that this action watches on certain vcluster requests and wants
// to mutate these. The objects this action wants to watch can be defined through the
// Resource() function that returns a new object of the type to watch. By implementing
// the defined interfaces below it is possible to watch on:
// Create, Update (includes patch requests), Delete and Get requests.
// This makes it possible to change incoming or outgoing objects on the fly, without the
// need to completely replace a vanilla vcluster syncer.
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
