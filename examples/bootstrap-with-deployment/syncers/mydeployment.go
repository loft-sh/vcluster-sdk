package syncers

import (
	"fmt"

	"github.com/loft-sh/vcluster-sdk/applier"
	"github.com/loft-sh/vcluster-sdk/log"
	"github.com/loft-sh/vcluster-sdk/syncer"
	synccontext "github.com/loft-sh/vcluster-sdk/syncer/context"
)

const (
	MydeploymentManifestPath = "/manifests/mydeployment.yaml"
)

func NewMydeploymentSyncer(ctx *synccontext.RegisterContext) syncer.Base {
	return &mydeploymentSyncer{}
}

type mydeploymentSyncer struct{}

var _ syncer.Base = &mydeploymentSyncer{}

func (s *mydeploymentSyncer) Name() string {
	return "mydeploymentsyncer"
}

var _ syncer.Initializer = &mydeploymentSyncer{}

func (s *mydeploymentSyncer) Init(ctx *synccontext.RegisterContext) error {
	err := applier.ApplyManifestFile(ctx.VirtualManager.GetConfig(), MydeploymentManifestPath)
	if err == nil {
		log.New(s.Name()).Infof("Successfully applied manifest %s", MydeploymentManifestPath)
	} else {
		err = fmt.Errorf("failed to apply manifest %s: %v", MydeploymentManifestPath, err)
	}
	return err
}
