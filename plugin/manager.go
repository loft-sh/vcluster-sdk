package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log/logr"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/plugin/types"
	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	"github.com/loft-sh/vcluster/pkg/setup/options"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	contextutil "github.com/loft-sh/vcluster/pkg/util/context"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func newManager() Manager {
	return &manager{}
}

type manager struct {
	m sync.Mutex

	context *synccontext.RegisterContext

	initialized bool
	started     bool

	syncerConfig clientcmd.ClientConfig

	pluginServer server

	syncers []syncertypes.Base
}

func (m *manager) UnmarshalConfig(into interface{}) error {
	err := yaml.Unmarshal([]byte(os.Getenv(v2.PluginConfigEnv)), into)
	if err != nil {
		return fmt.Errorf("unmarshal plugin config: %w", err)
	}

	return nil
}

func (m *manager) Init() (*synccontext.RegisterContext, error) {
	return m.InitWithOptions(Options{})
}

func (m *manager) InitWithOptions(opts Options) (*synccontext.RegisterContext, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if m.initialized {
		return nil, fmt.Errorf("plugin manager is already initialized")
	}
	m.initialized = true

	// create a new plugin server
	var err error
	m.pluginServer, err = newPluginServer()
	if err != nil {
		return nil, fmt.Errorf("create plugin server")
	}

	// serve plugin and block until we got the start info
	go m.pluginServer.Serve()

	// wait until we are started
	initRequest := <-m.pluginServer.Initialized()

	// decode init config
	initConfig := &v2.InitConfig{}
	err = json.Unmarshal([]byte(initRequest.Config), initConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding init config %s: %w", initRequest.Config, err)
	}

	// try to change working dir
	if initConfig.WorkingDir != "" {
		err = os.Chdir(initConfig.WorkingDir)
		if err != nil {
			return nil, fmt.Errorf("error changing working dir to %s: %w", initConfig.WorkingDir, err)
		}
	}

	// get current working dir
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// create logger and context
	logger, err := logr.NewLogger(currentWorkingDir)
	if err != nil {
		return nil, err
	}
	ctrl.SetLogger(logger)
	ctx := klog.NewContext(context.Background(), logger)

	// now create register context
	virtualClusterOptions := &options.VirtualClusterOptions{}
	err = json.Unmarshal(initConfig.Options, virtualClusterOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal vcluster options")
	}

	// set vcluster name correctly
	if virtualClusterOptions.Name != "" {
		translate.VClusterName = virtualClusterOptions.Name
	}

	// parse clients
	physicalConfig, err := clientcmd.NewClientConfigFromBytes(initConfig.PhysicalClusterConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse physical kube config")
	}
	restPhysicalConfig, err := physicalConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "parse physical kube config rest")
	}
	m.syncerConfig, err = clientcmd.NewClientConfigFromBytes(initConfig.SyncerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse syncer kube config")
	}

	// we disable plugin loading and create a new controller context
	virtualClusterOptions.DisablePlugins = true
	controllerCtx, err := setup.NewControllerContext(ctx, virtualClusterOptions, initConfig.CurrentNamespace, restPhysicalConfig, scheme.Scheme, opts.NewClient, opts.NewClient)
	if err != nil {
		return nil, fmt.Errorf("create controller context: %w", err)
	}

	m.context = contextutil.ToRegisterContext(controllerCtx)
	return m.context, nil
}

func (m *manager) Register(syncer syncertypes.Base) error {
	m.m.Lock()
	defer m.m.Unlock()

	m.syncers = append(m.syncers, syncer)
	return nil
}

func (m *manager) Start() error {
	err := m.start()
	if err != nil {
		return err
	}

	<-m.context.Context.Done()
	return nil
}

func (m *manager) start() error {
	m.m.Lock()
	defer m.m.Unlock()

	if m.started {
		return errors.New("plugin was already started")
	}
	m.started = true

	// find all hooks
	hooks, err := m.findAllHooks()
	if err != nil {
		return fmt.Errorf("find all hooks: %w", err)
	}

	// signal we are ready
	m.pluginServer.SetReady(hooks)

	// wait until we are leader to continue
	<-m.pluginServer.IsLeader()

	// start all syncers
	klog.Infof("Starting syncers...")
	for _, s := range m.syncers {
		initializer, ok := s.(syncertypes.Initializer)
		if ok {
			err := initializer.Init(m.context)
			if err != nil {
				return errors.Wrapf(err, "init syncer %s", s.Name())
			}
		}
	}
	for _, s := range m.syncers {
		indexRegisterer, ok := s.(syncertypes.IndicesRegisterer)
		if ok {
			err := indexRegisterer.RegisterIndices(m.context)
			if err != nil {
				return errors.Wrapf(err, "register indices for %s syncer", s.Name())
			}
		}
	}

	// start the local manager
	go func() {
		err := m.context.PhysicalManager.Start(m.context.Context)
		if err != nil {
			panic(err)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := m.context.VirtualManager.Start(m.context.Context)
		if err != nil {
			panic(err)
		}
	}()

	// Wait for caches to be synced
	m.context.PhysicalManager.GetCache().WaitForCacheSync(m.context.Context)
	m.context.VirtualManager.GetCache().WaitForCacheSync(m.context.Context)

	// set global owner
	err = setup.SetGlobalOwner(
		m.context.Context,
		m.context.CurrentNamespaceClient,
		m.context.CurrentNamespace,
		m.context.Options.TargetNamespace,
		m.context.Options.SetOwner,
		m.context.Options.ServiceName,
	)
	if err != nil {
		return fmt.Errorf("error in setting owner reference %v", err)
	}

	// start syncers
	for _, v := range m.syncers {
		// fake syncer?
		fakeSyncer, ok := v.(syncertypes.FakeSyncer)
		if ok {
			klog.Infof("Start fake syncer %s", fakeSyncer.Name())
			err = syncer.RegisterFakeSyncer(m.context, fakeSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}

		// real syncer?
		realSyncer, ok := v.(syncertypes.Syncer)
		if ok {
			klog.Infof("Start syncer %s", realSyncer.Name())
			err = syncer.RegisterSyncer(m.context, realSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}

		// controller starter?
		controllerStarter, ok := v.(syncertypes.ControllerStarter)
		if ok {
			klog.Infof("Start controller %s", v.Name())
			err = controllerStarter.Register(m.context)
			if err != nil {
				return errors.Wrapf(err, "start %s controller", v.Name())
			}
		}
	}

	klog.Infof("Successfully started plugin.")
	return nil
}

func (m *manager) findAllHooks() (map[types.VersionKindType][]ClientHook, error) {
	// gather all hooks
	hooks := map[types.VersionKindType][]ClientHook{}
	for _, s := range m.syncers {
		clientHook, ok := s.(ClientHook)
		if !ok {
			continue
		}

		obj := clientHook.Resource()
		gvk, err := clienthelper.GVKFrom(obj, scheme.Scheme)
		if err != nil {
			return nil, fmt.Errorf("cannot detect group version of resource object")
		}

		apiVersion, kind := gvk.ToAPIVersionAndKind()
		_, ok = clientHook.(MutateCreatePhysical)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "CreatePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateUpdatePhysical)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "UpdatePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateDeletePhysical)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "DeletePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateGetPhysical)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "GetPhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateCreateVirtual)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "CreateVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateUpdateVirtual)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "UpdateVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateDeleteVirtual)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "DeleteVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(MutateGetVirtual)
		if ok {
			apiVersionKindType := types.VersionKindType{
				APIVersion: apiVersion,
				Kind:       kind,
				Type:       "GetVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
	}

	return hooks, nil
}
