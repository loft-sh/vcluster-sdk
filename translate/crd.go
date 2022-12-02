package translate

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/loft-sh/vcluster-sdk/applier"
	"github.com/loft-sh/vcluster-sdk/log"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

func EnsureCRDFromPhysicalCluster(ctx context.Context, pConfig *rest.Config, vConfig *rest.Config, groupVersionKind schema.GroupVersionKind) error {
	exists, err := KindExists(vConfig, groupVersionKind)
	if err != nil {
		return errors.Wrap(err, "check virtual cluster kind")
	} else if exists {
		return nil
	}

	// get resource from kind name in physical cluster
	groupVersionResource, err := ConvertKindToResource(pConfig, groupVersionKind)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("seems like resource %s is not available in the physical cluster or vcluster has no access to it", groupVersionKind.String())
		}

		return err
	}

	// get crd in physical cluster
	pClient, err := apiextensionsv1clientset.NewForConfig(pConfig)
	if err != nil {
		return err
	}
	crdDefinition, err := pClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, groupVersionResource.GroupResource().String(), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieve crd in host cluster")
	}

	// now create crd in virtual cluster
	crdDefinition.UID = ""
	crdDefinition.ResourceVersion = ""
	crdDefinition.ManagedFields = nil
	crdDefinition.OwnerReferences = nil
	crdDefinition.Spec.PreserveUnknownFields = false
	crdDefinition.Status = apiextensionsv1.CustomResourceDefinitionStatus{}
	vClient, err := apiextensionsv1clientset.NewForConfig(vConfig)
	if err != nil {
		return err
	}

	log.NewWithoutName().Infof("Create crd %s in virtual cluster", groupVersionKind.String())
	_, err = vClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crdDefinition, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "create crd in virtual cluster")
	}

	// wait for crd to become ready
	log.NewWithoutName().Infof("Wait for crd %s to become ready in virtual cluster", groupVersionKind.String())
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		crdDefinition, err := vClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, groupVersionResource.GroupResource().String(), metav1.GetOptions{})
		if err != nil {
			return false, errors.Wrap(err, "retrieve crd in virtual cluster")
		}
		for _, cond := range crdDefinition.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1.Established:
				if cond.Status == apiextensionsv1.ConditionTrue {
					return true, nil
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for CRD %s to become ready: %v", groupVersionKind.String(), err)
	}

	return nil
}

func EnsureCRDFromFile(ctx context.Context, config *rest.Config, crdFilePath string, groupVersionKind schema.GroupVersionKind) error {
	exists, err := KindExists(config, groupVersionKind)
	if err != nil {
		return err
	} else if exists {
		return nil
	}

	log.NewWithoutName().Infof("Create crd %s in virtual cluster", groupVersionKind.String())
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: 5 * time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		err := applier.ApplyManifestFile(config, crdFilePath)
		if err != nil {
			log.NewWithoutName().Infof("Failed to apply CRD %s from the manifest file %s: %v", groupVersionKind.String(), crdFilePath, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to apply CRD %s: %v", groupVersionKind.String(), err)
	}

	var lastErr error
	log.NewWithoutName().Infof("Wait for crd %s to become ready in virtual cluster", groupVersionKind.String())
	err = wait.ExponentialBackoffWithContext(ctx, wait.Backoff{Duration: time.Second, Factor: 1.5, Cap: time.Minute, Steps: math.MaxInt32}, func() (bool, error) {
		var found bool
		found, lastErr = KindExists(config, groupVersionKind)
		return found, nil
	})
	if err != nil {
		return fmt.Errorf("failed to find CRD %s: %v: %v", groupVersionKind.String(), err, lastErr)
	}

	return nil
}

func ConvertKindToResource(config *rest.Config, groupVersionKind schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersionKind.GroupVersion().String())
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == groupVersionKind.Kind {
			return groupVersionKind.GroupVersion().WithResource(r.Name), nil
		}
	}

	return schema.GroupVersionResource{}, kerrors.NewNotFound(schema.GroupResource{Group: groupVersionKind.Group}, groupVersionKind.Kind)
}

// KindExists checks if given CRDs exist in the given group.
// Returns foundKinds, notFoundKinds, error
func KindExists(config *rest.Config, groupVersionKind schema.GroupVersionKind) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	resources, err := discoveryClient.ServerResourcesForGroupVersion(groupVersionKind.GroupVersion().String())
	if err != nil {
		if kerrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	for _, r := range resources.APIResources {
		if r.Kind == groupVersionKind.Kind {
			return true, nil
		}
	}

	return false, nil
}
