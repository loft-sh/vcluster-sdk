package mapper

import (
	"github.com/loft-sh/vcluster-sdk/syncer/context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reverse interface {
	GetReverseMapper() Config
	GetWatchers() []Watchers
}

type Config struct {
	ExtraIndices  []IndexFunc
	ExtraWatchers []Watchers
}

type Watchers func(ctx *context.RegisterContext, builder *builder.Builder) (*builder.Builder, error)

type IndexFunc func(ctx *context.RegisterContext) error

type NameCache map[types.NamespacedName]types.NamespacedName

type Enqueuer func(obj client.Object) []reconcile.Request
