package syncers

import (
	"fmt"

	"github.com/loft-sh/vcluster/pkg/syncer/synccontext"
	syncertypes "github.com/loft-sh/vcluster/pkg/syncer/types"
	"github.com/loft-sh/vcluster/pkg/util/applier"
	"k8s.io/klog/v2"
)

const (
	MyDeploymentManifestPath = "./manifests/mydeployment.yaml"
)

func NewMyDeploymentSyncer(ctx *synccontext.RegisterContext) syncertypes.Base {
	return &myDeploymentSyncer{}
}

type myDeploymentSyncer struct{}

var _ syncertypes.Base = &myDeploymentSyncer{}

func (s *myDeploymentSyncer) Name() string {
	return "mydeploymentsyncer"
}

var _ syncertypes.ControllerStarter = &myDeploymentSyncer{}

func (s *myDeploymentSyncer) Register(ctx *synccontext.RegisterContext) error {
	err := applier.ApplyManifestFile(ctx.Context, ctx.VirtualManager.GetConfig(), MyDeploymentManifestPath)
	if err != nil {
		return fmt.Errorf("failed to apply manifest %s: %w", MyDeploymentManifestPath, err)
	}

	klog.FromContext(ctx.Context).Info("Successfully applied manifest", "manifest", MyDeploymentManifestPath)
	return err
}
