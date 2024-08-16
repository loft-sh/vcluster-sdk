package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/ghodss/yaml"
	"github.com/loft-sh/log/logr"
	config2 "github.com/loft-sh/vcluster/config"
	"github.com/loft-sh/vcluster/pkg/config"
	"github.com/loft-sh/vcluster/pkg/controllers/syncer"
	synccontext "github.com/loft-sh/vcluster/pkg/controllers/syncer/context"
	"github.com/loft-sh/vcluster/pkg/plugin"
	"github.com/loft-sh/vcluster/pkg/plugin/types"
	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	"github.com/loft-sh/vcluster/pkg/scheme"
	"github.com/loft-sh/vcluster/pkg/setup"
	syncertypes "github.com/loft-sh/vcluster/pkg/types"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	contextutil "github.com/loft-sh/vcluster/pkg/util/context"
	"github.com/pkg/errors"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

func newManager() Manager {
	return &manager{
		interceptorsHandlers: make(map[string]http.Handler),
	}
}

type manager struct {
	m sync.Mutex

	context *synccontext.RegisterContext

	initialized bool
	started     bool

	syncerConfig clientcmd.ClientConfig

	pluginServer server

	syncers []syncertypes.Base

	interceptorsHandlers map[string]http.Handler
	interceptors         []Interceptor
	interceptorsPort     int

	proConfig v2.InitConfigPro

	options Options
}

func (m *manager) UnmarshalConfig(into interface{}) error {
	err := yaml.Unmarshal([]byte(os.Getenv(v2.PluginConfigEnv)), into)
	if err != nil {
		return fmt.Errorf("unmarshal plugin config: %w", err)
	}

	return nil
}

func (m *manager) ProConfig() v2.InitConfigPro {
	m.m.Lock()
	defer m.m.Unlock()

	return m.proConfig
}

func (m *manager) Init() (*synccontext.RegisterContext, error) {
	return m.InitWithOptions(Options{})
}

func (m *manager) InitWithOptions(options Options) (*synccontext.RegisterContext, error) {
	m.m.Lock()
	defer m.m.Unlock()

	if m.initialized {
		return nil, fmt.Errorf("plugin manager is already initialized")
	}
	m.initialized = true
	m.options = options

	// set code globals
	plugin.IsPlugin = true
	setup.NewLocalManager = m.newLocalManager
	setup.NewVirtualManager = m.newVirtualManager

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
	m.interceptorsPort = initConfig.Port

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
	virtualClusterConfig := &config.VirtualClusterConfig{}
	err = json.Unmarshal(initConfig.Config, virtualClusterConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal vCluster config")
	}

	// parse workload & control plane client
	virtualClusterConfig.WorkloadConfig, err = bytesToRestConfig(initConfig.WorkloadConfig)
	if err != nil {
		return nil, fmt.Errorf("parse workload config: %w", err)
	}
	virtualClusterConfig.ControlPlaneConfig, err = bytesToRestConfig(initConfig.ControlPlaneConfig)
	if err != nil {
		return nil, fmt.Errorf("parse control plane config: %w", err)
	}

	// we disable plugin loading
	virtualClusterConfig.Plugin = map[string]config2.Plugin{}
	virtualClusterConfig.Plugins = map[string]config2.Plugins{}

	// init virtual cluster config
	err = setup.InitAndValidateConfig(ctx, virtualClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("init config: %w", err)
	}

	// parse clients
	m.syncerConfig, err = clientcmd.NewClientConfigFromBytes(initConfig.SyncerConfig)
	if err != nil {
		return nil, errors.Wrap(err, "parse syncer kube config")
	}

	// create new controller context
	controllerCtx, err := setup.NewControllerContext(ctx, virtualClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("create controller context: %w", err)
	}

	m.context = contextutil.ToRegisterContext(controllerCtx)
	m.proConfig = initConfig.Pro
	return m.context, nil
}

func (m *manager) Register(syncer syncertypes.Base) error {
	m.m.Lock()
	defer m.m.Unlock()

	if int, ok := syncer.(Interceptor); ok {
		if _, ok := m.interceptorsHandlers[int.Name()]; ok {
			return fmt.Errorf("could not add the interceptor %s because its name is already in use", int.Name())
		}
		m.interceptorsHandlers[int.Name()] = int
		m.interceptors = append(m.interceptors, int)
	}

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

func (m *manager) StartAsync() (<-chan struct{}, error) {
	err := m.start()
	if err != nil {
		return nil, err
	}

	return m.context.Context.Done(), nil
}

func (m *manager) startInterceptors() error {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerName := r.Header.Get("VCluster-Plugin-Handler-Name")
		if handlerName == "" {
			responsewriters.InternalError(w, r, errors.New("header VCluster-Plugin-Handler-Name wasn't set"))
			return
		}
		interceptorHandler, ok := m.interceptorsHandlers[handlerName]
		if !ok {
			responsewriters.InternalError(w, r, errors.New("header VCluster-Plugin-Handler-Name had no match"))
			return
		}
		interceptorHandler.ServeHTTP(w, r)
	})

	return http.ListenAndServe("127.0.0.1:"+strconv.Itoa(m.interceptorsPort), handler)
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

	// find the interceptors
	interceptors := m.findAllInterceptors()

	// signal we are ready
	m.pluginServer.SetReady(hooks, interceptors, m.interceptorsPort)

	if len(m.interceptors) > 0 {
		go func() {
			// we need to start them regardless of being the leader, since the traffic is
			// directed to all replicas
			err := m.startInterceptors()
			if err != nil {
				klog.Error(err, "error while running the http interceptors:")
				os.Exit(1)
			}
		}()
	}
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
			klog.Errorf("Starting physical manager: %v", err)
			Exit(1)
		}
	}()

	// start the virtual cluster manager
	go func() {
		err := m.context.VirtualManager.Start(m.context.Context)
		if err != nil {
			klog.Errorf("Starting virtual manager: %v", err)
			Exit(1)
		}
	}()

	// Wait for caches to be synced
	m.context.PhysicalManager.GetCache().WaitForCacheSync(m.context.Context)
	m.context.VirtualManager.GetCache().WaitForCacheSync(m.context.Context)

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

func (m *manager) findAllInterceptors() []Interceptor {
	klog.Info("len of m.interceptor is : ", len(m.interceptors))
	return m.interceptors
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

func (m *manager) newLocalManager(config *rest.Config, options ctrlmanager.Options) (ctrlmanager.Manager, error) {
	options.Metrics.BindAddress = "0"
	if m.options.ModifyHostManager != nil {
		m.options.ModifyHostManager(&options)
	}

	return ctrlmanager.New(config, options)
}

func (m *manager) newVirtualManager(config *rest.Config, options ctrlmanager.Options) (ctrlmanager.Manager, error) {
	options.Metrics.BindAddress = "0"
	if m.options.ModifyVirtualManager != nil {
		m.options.ModifyVirtualManager(&options)
	}

	return ctrlmanager.New(config, options)
}

func bytesToRestConfig(rawBytes []byte) (*rest.Config, error) {
	if len(rawBytes) == 0 {
		return nil, fmt.Errorf("kube client config is empty")
	}

	parsedConfig, err := clientcmd.NewClientConfigFromBytes(rawBytes)
	if err != nil {
		return nil, fmt.Errorf("parse kube config: %w", err)
	}

	return parsedConfig.ClientConfig()
}
