package plugin

import (
	"context"
	"net/http"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

type Options struct {
	// ModifyVirtualManager modifies options for the virtual manager
	ModifyVirtualManager func(options *ctrlmanager.Options)

	// ModifyHostManager modifies options for the host manager
	ModifyHostManager func(options *ctrlmanager.Options)
}

type Manager interface {
	// Init creates a new plugin context and will block until the
	// vcluster container instance could be contacted.
	Init() (*synccontext.RegisterContext, error)

	// InitWithOptions creates a new plugin context and will block until the
	// vcluster container instance could be contacted.
	InitWithOptions(opts Options) (*synccontext.RegisterContext, error)

	// Register makes sure the syncer will be executed as soon as start
	// is run.
	Register(syncer syncertypes.Base) error

	// Start runs all the registered syncers and will block. It only executes
	// the functionality if the current vcluster pod is the current leader and
	// will stop if the pod will lose leader election.
	Start() error

	// Start runs all the registered syncers and will not block. It only executes
	// the functionality if the current vcluster pod is the current leader
	// You need to exit the plugin when the channel is closed since that means you lost
	// leader election
	StartAsync() (<-chan struct{}, error)

	// UnmarshalConfig retrieves the plugin config from environment and parses it into
	// the given object.
	UnmarshalConfig(into interface{}) error

	// ProConfig returns the pro config retrieved by vCluster.Pro
	ProConfig() v2.InitConfigPro
}

// ClientHook tells the sdk that this action watches on certain vcluster requests and wants
// to mutate these. The objects this action wants to watch can be defined through the
// Resource() function that returns a new object of the type to watch. By implementing
// the defined interfaces below it is possible to watch on:
// Create, Update (includes patch requests), Delete and Get requests.
// This makes it possible to change incoming or outgoing objects on the fly, without the
// need to completely replace a vanilla vcluster syncer.
type ClientHook interface {
	syncertypes.Base

	// Resource is the typed resource (e.g. &corev1.Pod{}) that should get mutated.
	Resource() client.Object
}

type Interceptor interface {
	syncertypes.Base

	// Handler is the handler that will handle the requests delegated by the syncer
	http.Handler

	// InterceptionRules returns an rbac style struct which defines what to intercept
	InterceptionRules() []v2.InterceptorRule
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
