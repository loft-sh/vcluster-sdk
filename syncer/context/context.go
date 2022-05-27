package context

import (
	"context"
	"github.com/loft-sh/vcluster-sdk/log"
	"k8s.io/client-go/tools/clientcmd"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncContext is the context used in syncer functions that
// interact with the virtual and physical cluster after the
// controller has been started.
type SyncContext struct {
	// Context is the golang context used to create the managers
	Context context.Context

	// Log is a prefixed log with the syncer name
	Log log.Logger

	// TargetNamespace is the namespace in the host cluster where
	// vcluster should sync namespaced scope objects to
	TargetNamespace string

	// PhysicalClient is a cached client that can be used to retrieve
	// and modify objects in the target namespace as well as cluster
	// scoped objects.
	PhysicalClient client.Client

	// VirtualClient is a cached client that can be used to retrieve
	// and modify objects in the virtual backing cluster.
	VirtualClient client.Client

	// CurrentNamespace is the namespace vcluster including this plugin
	// is currently running in.
	CurrentNamespace string

	// CurrentNamespaceClient is a cached client that can be used to retrieve
	// and modify objects in the current namespace where vcluster is running
	// in. Should not be used to sync virtual cluster objects.
	CurrentNamespaceClient client.Client
}

// RegisterContext is the context used to register additional
// indices or modify the syncer controller and provides direct
// access to the underlying controller runtime managers.
type RegisterContext struct {
	// Context is the golang context used to create the managers
	Context context.Context

	// Options holds the vcluster flags that were used to start the vcluster
	Options *VirtualClusterOptions

	// TargetNamespace is the namespace in the host cluster where
	// vcluster should sync namespaced scope objects to
	TargetNamespace string

	// CurrentNamespace is the namespace vcluster including this plugin
	// is currently running in.
	CurrentNamespace string

	// CurrentNamespaceClient is a cached client that can be used to retrieve
	// and modify objects in the current namespace where vcluster is running
	// in. Should not be used to sync virtual cluster objects.
	CurrentNamespaceClient client.Client

	// VirtualManager is the controller runtime manager connected directly to the
	// backing virtual cluster, such as k0s, k3s or vanilla k8s.
	VirtualManager ctrl.Manager

	// PhysicalManager is the controller runtime manager connected directly to the
	// host cluster namespace where vcluster will sync to.
	PhysicalManager ctrl.Manager

	// SyncerConfig can be used to perform requests directly to the vcluster syncer
	// as other applications outside or inside vcluster would do. This client differs from
	// virtual and physical manager, as these are either connecting to the host or backing
	// virtual cluster directly.
	SyncerConfig clientcmd.ClientConfig
}

// VirtualClusterOptions holds the vcluster flags that were used to start the vcluster
// VirtualClusterOptions holds the cmd flags
type VirtualClusterOptions struct {
	Controllers []string `json:"controllers,omitempty"`

	ServerCaCert        string   `json:"serverCaCert,omitempty"`
	ServerCaKey         string   `json:"serverCaKey,omitempty"`
	TLSSANs             []string `json:"tlsSans,omitempty"`
	RequestHeaderCaCert string   `json:"requestHeaderCaCert,omitempty"`
	ClientCaCert        string   `json:"clientCaCert,omitempty"`
	KubeConfig          string   `json:"kubeConfig,omitempty"`

	KubeConfigSecret          string   `json:"kubeConfigSecret,omitempty"`
	KubeConfigSecretNamespace string   `json:"kubeConfigSecretNamespace,omitempty"`
	KubeConfigServer          string   `json:"kubeConfigServer,omitempty"`
	Tolerations               []string `json:"tolerations,omitempty"`

	BindAddress string `json:"bindAddress,omitempty"`
	Port        int    `json:"port,omitempty"`

	Name string `json:"name,omitempty"`

	TargetNamespace string `json:"targetNamespace,omitempty"`
	ServiceName     string `json:"serviceName,omitempty"`

	SetOwner bool `json:"setOwner,omitempty"`

	SyncAllNodes        bool `json:"syncAllNodes,omitempty"`
	EnableScheduler     bool `json:"enableScheduler,omitempty"`
	DisableFakeKubelets bool `json:"disableFakeKubelets,omitempty"`

	TranslateImages []string `json:"translateImages,omitempty"`

	NodeSelector        string `json:"nodeSelector,omitempty"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector,omitempty"`
	ServiceAccount      string `json:"serviceAccount,omitempty"`

	OverrideHosts               bool   `json:"overrideHosts,omitempty"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage,omitempty"`

	ClusterDomain string `json:"clusterDomain,omitempty"`

	LeaderElect   bool  `json:"leaderElect,omitempty"`
	LeaseDuration int64 `json:"leaseDuration,omitempty"`
	RenewDeadline int64 `json:"renewDeadline,omitempty"`
	RetryPeriod   int64 `json:"retryPeriod,omitempty"`

	DisablePlugins      bool     `json:"disablePlugins,omitempty"`
	PluginListenAddress string   `json:"pluginListenAddress,omitempty"`
	Plugins             []string `json:"plugins,omitempty"`

	DefaultImageRegistry string `json:"defaultImageRegistry,omitempty"`

	EnforcePodSecurityStandard string `json:"enforcePodSecurityStandard,omitempty"`

	MapHostServices    []string `json:"mapHostServices,omitempty"`
	MapVirtualServices []string `json:"mapVirtualServices,omitempty"`

	SyncLabels []string `json:"syncLabels,omitempty"`

	// DEPRECATED FLAGS
	DeprecatedSyncNodeChanges          bool `json:"syncNodeChanges"`
	DeprecatedDisableSyncResources     string
	DeprecatedOwningStatefulSet        string
	DeprecatedUseFakeNodes             bool
	DeprecatedUseFakePersistentVolumes bool
	DeprecatedEnableStorageClasses     bool
	DeprecatedEnablePriorityClasses    bool
	DeprecatedSuffix                   string
	DeprecatedUseFakeKubelets          bool
}

func ConvertContext(registerContext *RegisterContext, logName string) *SyncContext {
	return &SyncContext{
		Context:                registerContext.Context,
		Log:                    log.New(logName),
		TargetNamespace:        registerContext.TargetNamespace,
		PhysicalClient:         registerContext.PhysicalManager.GetClient(),
		VirtualClient:          registerContext.VirtualManager.GetClient(),
		CurrentNamespace:       registerContext.CurrentNamespace,
		CurrentNamespaceClient: registerContext.CurrentNamespaceClient,
	}
}
