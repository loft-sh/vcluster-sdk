package plugin

import (
	"os"
	"time"

	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"k8s.io/klog/v2"
)

var (
	defaultManager = newManager()
)

func MustInit() *synccontext.RegisterContext {
	ctx, err := defaultManager.Init()
	if err != nil {
		klog.Errorf("plugin must init: %v", err)
		Exit(1)
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
		klog.Errorf("plugin must register: %v", err)
		Exit(1)
	}
}

func Register(syncer syncertypes.Base) error {
	return defaultManager.Register(syncer)
}

func MustStart() {
	err := defaultManager.Start()
	if err != nil {
		klog.Errorf("plugin must start: %v", err)
		Exit(1)
	}
}

func Start() error {
	return defaultManager.Start()
}

func StartAsync() (<-chan struct{}, error) {
	return defaultManager.StartAsync()
}

func UnmarshalConfig(into interface{}) error {
	return defaultManager.UnmarshalConfig(into)
}

func ProConfig() v2.InitConfigPro { return defaultManager.ProConfig() }

func Exit(code int) {
	// we need to wait here or else we won't see a message
	time.Sleep(time.Millisecond * 500)
	os.Exit(code)
}
