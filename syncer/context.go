package syncer

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Context holds the plugin context
type Context struct {
	// Context is the context to use for requests
	Context context.Context

	// PhysicalClient is a cache client that points to the
	// physical host cluster namespace. Requests are scoped to the
	// host namespace. Cluster scoped objects are not affected from this.
	PhysicalClient client.Client

	// VirtualClient is a cache client that points to the virtual cluster
	// Requests are not scoped to any namespace and the client has full cluster
	// wide access.
	VirtualClient client.Client

	// TargetNamespace is the namespace vcluster will sync objects to. This is usually the same
	// namespace as CurrentNamespace, but could be different based on the vcluster start options.
	TargetNamespace string

	// CurrentNamespace is the namespace vcluster is currently running in. This usually points
	// to the same namespace as TargetNamespace, but could be different based on the vcluster
	// start options.
	CurrentNamespace string

	// CurrentNamespaceClient
	CurrentNamespaceClient client.Client
}
