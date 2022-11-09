package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/loft-sh/vcluster-sdk/clienthelper"
	"github.com/loft-sh/vcluster-sdk/hook"
	"github.com/loft-sh/vcluster-sdk/log"
	"github.com/loft-sh/vcluster-sdk/plugin/remote"
	"github.com/loft-sh/vcluster-sdk/syncer"
	synccontext "github.com/loft-sh/vcluster-sdk/syncer/context"
	"github.com/loft-sh/vcluster-sdk/translate"
	"github.com/loft-sh/vcluster-sdk/translate/util"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

const (
	PLUGIN_SERVER_ADDRESS = "VCLUSTER_PLUGIN_ADDRESS"
	PLUGIN_NAME           = "VCLUSTER_PLUGIN_NAME"
)

var defaultManager Manager = &manager{}

type Options struct {
	// ListenAddress is optional and the address where to contact
	// the vcluster plugin server at. Defaults to localhost:10099
	ListenAddress string

	// NewClient is the func that creates the client to be used by the manager.
	// If not set this will create the default DelegatingClient that will
	// use the cache for reads and the client for writes.
	NewClient cluster.NewClientFunc

	// NewCache is the function that will create the cache to be used
	// by the manager. If not set this will use the default new cache function.
	NewCache cache.NewCacheFunc
}

type LeaderElectionHook func(ctx context.Context) error

type Manager interface {
	// Init creates a new plugin context and will block until the
	// vcluster container instance could be contacted.
	Init() (*synccontext.RegisterContext, error)

	// InitWithOptions creates a new plugin context and will block until the
	// vcluster container instance could be contacted.
	InitWithOptions(opts Options) (*synccontext.RegisterContext, error)

	// Register makes sure the syncer will be executed as soon as start
	// is run.
	Register(syncer syncer.Base) error

	// Start runs all the registered syncers and will block. It only executes
	// the functionality if the current vcluster pod is the current leader and
	// will stop if the pod will lose leader election.
	Start() error
}

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

func MustRegister(syncer syncer.Base) {
	err := defaultManager.Register(syncer)
	if err != nil {
		panic(err)
	}
}

func Register(syncer syncer.Base) error {
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

type manager struct {
	guard       sync.Mutex
	initialized bool
	started     bool
	syncers     []syncer.Base

	name    string
	address string
	context *synccontext.RegisterContext

	hooks []*remote.ClientHook
}

func (m *manager) Init() (*synccontext.RegisterContext, error) {
	return m.InitWithOptions(Options{})
}

func (m *manager) InitWithOptions(opts Options) (*synccontext.RegisterContext, error) {
	name := os.Getenv(PLUGIN_NAME)
	if name == "" {
		return nil, fmt.Errorf("plugin name wasn't found in environment. vcluster-sdk expects the vcluster to set the %s environment variable", PLUGIN_NAME)
	}

	m.guard.Lock()
	defer m.guard.Unlock()

	if m.initialized {
		return nil, fmt.Errorf("plugin manager is already initialized")
	}
	m.initialized = true

	log := log.New("plugin")
	m.name = name
	m.address = "localhost:10099"
	if opts.ListenAddress != "" {
		m.address = opts.ListenAddress
	}

	log.Infof("Try creating context...")
	var pluginContext *remote.Context
	err := wait.PollImmediateInfinite(time.Second*5, func() (done bool, err error) {
		conn, err := grpc.Dial(m.address, grpc.WithInsecure())
		if err != nil {
			return false, nil
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		pluginContext, err = remote.NewVClusterClient(conn).GetContext(ctx, &remote.Empty{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	// now create register context
	virtualClusterOptions := &synccontext.VirtualClusterOptions{}
	err = json.Unmarshal([]byte(pluginContext.Options), virtualClusterOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal vcluster options")
	}

	// set vcluster name correctly
	if virtualClusterOptions.Name != "" {
		translate.Suffix = virtualClusterOptions.Name
	}

	// TODO: support find owner

	// parse clients
	physicalConfig, err := clientcmd.NewClientConfigFromBytes([]byte(pluginContext.PhysicalClusterConfig))
	if err != nil {
		return nil, errors.Wrap(err, "parse physical kube config")
	}
	restPhysicalConfig, err := physicalConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "parse physical kube config rest")
	}
	virtualConfig, err := clientcmd.NewClientConfigFromBytes([]byte(pluginContext.VirtualClusterConfig))
	if err != nil {
		return nil, errors.Wrap(err, "parse virtual kube config")
	}
	restVirtualConfig, err := virtualConfig.ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "parse virtual kube config rest")
	}
	syncerConfig, err := clientcmd.NewClientConfigFromBytes([]byte(pluginContext.SyncerConfig))
	if err != nil {
		return nil, errors.Wrap(err, "parse syncer kube config")
	}

	// We increase the limits here so that we don't get any problems
	restVirtualConfig.QPS = 1000
	restVirtualConfig.Burst = 2000
	restVirtualConfig.Timeout = 0

	restPhysicalConfig.QPS = 40
	restPhysicalConfig.Burst = 80
	restPhysicalConfig.Timeout = 0

	physicalManager, err := ctrl.NewManager(restPhysicalConfig, ctrl.Options{
		Scheme:             Scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		Namespace:          virtualClusterOptions.TargetNamespace,
		NewClient:          opts.NewClient,
		NewCache:           opts.NewCache,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create phyiscal manager")
	}
	virtualManager, err := ctrl.NewManager(restVirtualConfig, ctrl.Options{
		Scheme:             Scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
		NewClient:          opts.NewClient,
		NewCache:           opts.NewCache,
	})
	if err != nil {
		return nil, errors.Wrap(err, "create virtual manager")
	}
	currentNamespaceClient, err := newCurrentNamespaceClient(context.Background(), pluginContext.CurrentNamespace, physicalManager, virtualClusterOptions, opts)
	if err != nil {
		return nil, errors.Wrap(err, "create namespaced client")
	}

	m.context = &synccontext.RegisterContext{
		Context:                context.Background(),
		Options:                virtualClusterOptions,
		TargetNamespace:        pluginContext.TargetNamespace,
		CurrentNamespace:       pluginContext.CurrentNamespace,
		CurrentNamespaceClient: currentNamespaceClient,
		VirtualManager:         virtualManager,
		PhysicalManager:        physicalManager,
		SyncerConfig:           syncerConfig,
	}
	return m.context, nil
}

func (m *manager) Register(syncer syncer.Base) error {
	m.guard.Lock()
	defer m.guard.Unlock()
	if m.started {
		return fmt.Errorf("plugin manager already started")
	}

	m.syncers = append(m.syncers, syncer)
	return nil
}

func (m *manager) Start() error {
	m.guard.Lock()
	err := m.start()
	m.guard.Unlock()
	if err != nil {
		return err
	}

	<-m.context.Context.Done()
	return nil
}

func (m *manager) registerPlugin(log log.Logger) error {
	serverAddress := os.Getenv(PLUGIN_SERVER_ADDRESS)
	if serverAddress == "" {
		log.Errorf("Environment variable %s not defined, are you using an old vcluster version?", PLUGIN_SERVER_ADDRESS)
		return nil
	}

	// gather all hooks
	hooks := map[ApiVersionKindType][]hook.ClientHook{}
	for _, s := range m.syncers {
		clientHook, ok := s.(hook.ClientHook)
		if !ok {
			continue
		}

		obj := clientHook.Resource()
		gvk, err := clienthelper.GVKFrom(obj, Scheme)
		if err != nil {
			return fmt.Errorf("cannot detect group version of resource object")
		}

		apiVersion, kind := gvk.ToAPIVersionAndKind()
		_, ok = clientHook.(hook.MutateCreatePhysical)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "CreatePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateUpdatePhysical)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "UpdatePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateDeletePhysical)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "DeletePhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateGetPhysical)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "GetPhysical",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateCreateVirtual)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "CreateVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateUpdateVirtual)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "UpdateVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateDeleteVirtual)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "DeleteVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
		_, ok = clientHook.(hook.MutateGetVirtual)
		if ok {
			apiVersionKindType := ApiVersionKindType{
				ApiVersion: apiVersion,
				Kind:       kind,
				Type:       "GetVirtual",
			}
			hooks[apiVersionKindType] = append(hooks[apiVersionKindType], clientHook)
		}
	}

	// transform hooks
	registeredHooks := []*remote.ClientHook{}
	for key := range hooks {
		hookFound := false
		for _, h := range registeredHooks {
			if h.ApiVersion == key.ApiVersion && h.Kind == key.Kind {
				found := false
				for _, t := range h.Types {
					if t == key.Type {
						found = true
						break
					}
				}
				if !found {
					h.Types = append(h.Types, key.Type)
				}

				hookFound = true
				break
			}
		}

		if !hookFound {
			registeredHooks = append(registeredHooks, &remote.ClientHook{
				ApiVersion: key.ApiVersion,
				Kind:       key.Kind,
				Types:      []string{key.Type},
			})
		}
	}

	// start the grpc server
	if len(registeredHooks) > 0 {
		log.Infof("Plugin server listening on %s", serverAddress)
		lis, err := net.Listen("tcp", serverAddress)
		if err != nil {
			return fmt.Errorf("failed to listen: %v", err)
		}

		var opts []grpc.ServerOption
		grpcServer := grpc.NewServer(opts...)
		remote.RegisterPluginServer(grpcServer, &pluginServer{
			hooks:           hooks,
			registeredHooks: registeredHooks,
		})
		go func() {
			err := grpcServer.Serve(lis)
			if err != nil {
				panic(err)
			}
		}()
	}

	// register the plugin
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	conn, err := grpc.Dial(m.address, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("error dialing vcluster: %v", err)
	}

	defer conn.Close()
	_, err = remote.NewVClusterClient(conn).RegisterPlugin(ctx, &remote.RegisterPluginRequest{
		Version:     "v1",
		Name:        m.name,
		Address:     serverAddress,
		ClientHooks: registeredHooks,
	})
	if err != nil {
		log.Errorf("error trying to connect to vcluster: %v", err)
		return err
	}

	return nil
}

func (m *manager) start() error {
	log := log.New("plugin")
	if m.started {
		return fmt.Errorf("manager was already started")
	}

	err := m.registerPlugin(log)
	if err != nil {
		return errors.Wrap(err, "start hook server")
	}

	// runID is used to track if the syncer was restarted while we are listening
	runID := ""

	log.Infof("Waiting for vcluster to become leader...")
	conn, err := grpc.Dial(m.address, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("error dialing vcluster: %v", err)
	}
	defer conn.Close()
	err = wait.PollImmediateInfinite(time.Second*5, func() (done bool, err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		isLeader, err := remote.NewVClusterClient(conn).IsLeader(ctx, &remote.Empty{})
		if err != nil {
			log.Errorf("error trying to connect to vcluster: %v", err)
			conn.Close()
			conn, err = grpc.Dial(m.address, grpc.WithInsecure())
			if err != nil {
				return false, err
			}
			return false, nil
		}

		if runID == "" {
			runID = isLeader.RunID
		} else if runID != isLeader.RunID {
			return false, fmt.Errorf("syncer runID has changed while starting up, restarting plugin")
		}

		return isLeader.Leader, nil
	})
	if err != nil {
		return err
	}

	// keep watching for run id
	go func() {
		conn, err := grpc.Dial(m.address, grpc.WithInsecure())
		if err != nil {
			klog.Fatalf("error dialing vcluster: %v", err)
		}
		defer conn.Close()

		err = wait.PollInfinite(time.Second*5, func() (done bool, err error) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()

			isLeader, err := remote.NewVClusterClient(conn).IsLeader(ctx, &remote.Empty{})
			if err != nil {
				log.Errorf("error trying to connect to vcluster: %v", err)
				conn.Close()
				conn, err = grpc.Dial(m.address, grpc.WithInsecure())
				if err != nil {
					return false, err
				}
				return false, nil
			}

			if runID != isLeader.RunID {
				return false, fmt.Errorf("syncer has restarted, restarting plugin")
			} else if !isLeader.Leader {
				return false, fmt.Errorf("syncer lost leader election, restarting plugin")
			}

			return false, nil
		})
		if err != nil {
			klog.Fatalf("error watching syncer: %v", err)
		}
	}()

	m.started = true
	log.Infof("Starting syncers...")
	for _, s := range m.syncers {
		initializer, ok := s.(syncer.Initializer)
		if ok {
			err := initializer.Init(m.context)
			if err != nil {
				return errors.Wrapf(err, "init syncer %s", s.Name())
			}
		}
	}
	for _, s := range m.syncers {
		indexRegisterer, ok := s.(syncer.IndicesRegisterer)
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

	err = util.FindOwner(m.context)
	if err != nil {
		return fmt.Errorf("error in setting owner reference %v", err)
	}
	// start syncers
	for _, v := range m.syncers {
		// fake syncer?
		fakeSyncer, ok := v.(syncer.FakeSyncer)
		if ok {
			log.Infof("Start fake syncer %s", fakeSyncer.Name())
			err = syncer.RegisterFakeSyncer(m.context, fakeSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}

		// real syncer?
		realSyncer, ok := v.(syncer.Syncer)
		if ok {
			log.Infof("Start syncer %s", realSyncer.Name())
			err = syncer.RegisterSyncer(m.context, realSyncer)
			if err != nil {
				return errors.Wrapf(err, "start %s syncer", v.Name())
			}
		}

		// controller starter?
		controllerStarter, ok := v.(syncer.ControllerStarter)
		if ok {
			log.Infof("Start controller %s", v.Name())
			err = controllerStarter.Register(m.context)
			if err != nil {
				return errors.Wrapf(err, "start %s controller", v.Name())
			}
		}
	}

	log.Infof("Successfully started plugin.")
	return nil
}

func newCurrentNamespaceClient(ctx context.Context, currentNamespace string, localManager ctrl.Manager, options *synccontext.VirtualClusterOptions, opts Options) (client.Client, error) {
	var err error

	// currentNamespaceCache is needed for tasks such as finding out fake kubelet ips
	// as those are saved as Kubernetes services inside the same namespace as vcluster
	// is running. In the case of options.TargetNamespace != currentNamespace (the namespace
	// where vcluster is currently running in), we need to create a new object cache
	// as the regular cache is scoped to the options.TargetNamespace and cannot return
	// objects from the current namespace.
	currentNamespaceCache := localManager.GetCache()
	if currentNamespace != options.TargetNamespace {
		if opts.NewCache == nil {
			opts.NewCache = cache.New
		}

		currentNamespaceCache, err = opts.NewCache(localManager.GetConfig(), cache.Options{
			Scheme:    localManager.GetScheme(),
			Mapper:    localManager.GetRESTMapper(),
			Namespace: currentNamespace,
		})
		if err != nil {
			return nil, err
		}
	}

	// start cache now if it's not in the same namespace
	if currentNamespace != options.TargetNamespace {
		go func() {
			err := currentNamespaceCache.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		currentNamespaceCache.WaitForCacheSync(ctx)
	}

	// create a current namespace client
	if opts.NewClient == nil {
		opts.NewClient = cluster.DefaultNewClient
	}

	return opts.NewClient(currentNamespaceCache, localManager.GetConfig(), client.Options{
		Scheme: localManager.GetScheme(),
		Mapper: localManager.GetRESTMapper(),
	})
}
