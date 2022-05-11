package hooks

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster-sdk/hook"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewServiceHook() hook.ClientHook {
	return &serviceHook{}
}

// Purpose of this hook is to add an extra port to each service
type serviceHook struct{}

func (s *serviceHook) Name() string {
	return "service-hook"
}

func (s *serviceHook) Resource() client.Object {
	return &corev1.Service{}
}

var _ hook.MutateCreatePhysical = &serviceHook{}

func (s *serviceHook) MutateCreatePhysical(ctx context.Context, obj client.Object) (client.Object, error) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("object %v is not a service", obj)
	}

	addPort(service)
	return service, nil
}

var _ hook.MutateGetVirtual = &serviceHook{}

// MutateGetVirtual fakes the service vcluster "sees" so that it is not trying to update the
// ports all the time
func (s *serviceHook) MutateGetVirtual(ctx context.Context, obj client.Object) (client.Object, error) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		return nil, fmt.Errorf("object %v is not a service", obj)
	}

	addPort(service)
	return service, nil
}

func addPort(service *corev1.Service) {
	found := false
	for _, p := range service.Spec.Ports {
		if p.Name == "plugin" && p.Protocol == corev1.ProtocolTCP && p.Port == 19000 {
			found = true
			break
		}
	}
	if !found {
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     "plugin",
			Protocol: corev1.ProtocolTCP,
			Port:     19000,
		})
	}
}
