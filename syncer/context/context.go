package context

import (
	"context"
	"github.com/loft-sh/vcluster-sdk/log"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncContext is the context used in syncer functions that
// interact with the virtual and physical cluster after the
// controller has been started.
type SyncContext struct {
	Context context.Context
	Log     log.Logger

	TargetNamespace string
	PhysicalClient  client.Client

	VirtualClient client.Client

	CurrentNamespace       string
	CurrentNamespaceClient client.Client
}

// RegisterContext is the context used to register additional
// indices or modify the syncer controller and provides direct
// access to the underlying controller runtime managers.
type RegisterContext struct {
	Context          context.Context
	EventBroadcaster record.EventBroadcaster

	Options *VirtualClusterOptions

	TargetNamespace        string
	CurrentNamespace       string
	CurrentNamespaceClient client.Client

	VirtualManager  ctrl.Manager
	PhysicalManager ctrl.Manager

	SyncerConfig clientcmd.ClientConfig
}

// VirtualClusterOptions holds the vcluster flags
type VirtualClusterOptions struct {
	Controllers string `json:"controllers,omitempty"`

	ServerCaCert        string   `json:"serverCaCert,omitempty"`
	ServerCaKey         string   `json:"serverCaKey,omitempty"`
	TLSSANs             []string `json:"tlsSans,omitempty"`
	RequestHeaderCaCert string   `json:"requestHeaderCaCert,omitempty"`
	ClientCaCert        string   `json:"clientCaCert"`
	KubeConfig          string   `json:"kubeConfig"`

	KubeConfigSecret          string `json:"kubeConfigSecret"`
	KubeConfigSecretNamespace string `json:"kubeConfigSecretNamespace"`
	KubeConfigServer          string `json:"kubeConfigServer"`

	BindAddress string `json:"bindAddress"`
	Port        int    `json:"port"`

	Name string `json:"name"`

	TargetNamespace string `json:"targetNamespace"`
	ServiceName     string `json:"serviceName"`

	SetOwner bool `json:"setOwner"`

	SyncAllNodes        bool `json:"syncAllNodes"`
	SyncNodeChanges     bool `json:"syncNodeChanges"`
	DisableFakeKubelets bool `json:"disableFakeKubelets"`

	TranslateImages []string `json:"translateImages"`

	NodeSelector        string `json:"nodeSelector"`
	ServiceAccount      string `json:"serviceAccount"`
	EnforceNodeSelector bool   `json:"enforceNodeSelector"`

	OverrideHosts               bool   `json:"overrideHosts"`
	OverrideHostsContainerImage string `json:"overrideHostsContainerImage"`

	ClusterDomain string `json:"clusterDomain"`

	LeaderElect   bool  `json:"leaderElect"`
	LeaseDuration int64 `json:"leaseDuration"`
	RenewDeadline int64 `json:"renewDeadline"`
	RetryPeriod   int64 `json:"retryPeriod"`

	DisablePlugins      bool   `json:"disablePlugins"`
	PluginListenAddress string `json:"pluginListenAddress"`

	// DEPRECATED FLAGS
	DeprecatedDisableSyncResources     string
	DeprecatedOwningStatefulSet        string
	DeprecatedUseFakeNodes             bool
	DeprecatedUseFakePersistentVolumes bool
	DeprecatedEnableStorageClasses     bool
	DeprecatedEnablePriorityClasses    bool
	DeprecatedSuffix                   string
	DeprecatedUseFakeKubelets          bool
}
