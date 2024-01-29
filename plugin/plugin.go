package plugin

import (
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
)

var (
	defaultManager = newManager()
)

func MustInit() *synccontext.RegisterContext {
	ctx, err := defaultManager.Init()
	if err != nil {
		panic(err)
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
		panic(err)
	}
}

func Register(syncer syncertypes.Base) error {
	return defaultManager.Register(syncer)
}

func MustStart() {
	err := defaultManager.Start()
	if err != nil {
		panic(err)
	}
}

func Start() error {
	return defaultManager.Start()
}
