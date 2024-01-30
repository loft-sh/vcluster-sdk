package plugin

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"k8s.io/klog/v2"
)

var (
	defaultManager = newManager()
)

func MustInit() *synccontext.RegisterContext {
	ctx, err := defaultManager.Init()
	if err != nil {
		klog.Fatalf("plugin must init: %v", err)
	}

	return ctx
}

func Init() (*synccontext.RegisterContext, error) {
	return defaultManager.Init()
}

func InitWithOptions(opts Options) (*synccontext.RegisterContext, error) {
	return defaultManager.InitWithOptions(opts)
}

func MustRegister(syncer syncertypes.Base) {
	err := defaultManager.Register(syncer)
	if err != nil {
		klog.Fatalf("plugin must register: %v", err)
	}
}

func Register(syncer syncertypes.Base) error {
	return defaultManager.Register(syncer)
}

func MustStart() {
	err := defaultManager.Start()
	if err != nil {
		klog.Fatalf("plugin must start: %v", err)
	}
}

func Start() error {
	return defaultManager.Start()
}

func UnmarshalConfig(into interface{}) error {
	return defaultManager.UnmarshalConfig(into)
}
