package plugin

import (
	"time"

	examplev1 "github.com/loft-sh/vcluster-sdk/e2e/test_plugin/apis/v1"
	"github.com/loft-sh/vcluster/pkg/util/translate"
	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	pollingInterval     = time.Second * 2
	pollingDurationLong = time.Second * 60
)

var _ = ginkgo.Describe("Plugin test", func() {
	f := framework.DefaultFramework

	ginkgo.It("check virtual deployment is there", func() {
		// wait for virtual deployment to be deployed
		var err error
		var virtualDeployments *appsv1.DeploymentList
		gomega.Eventually(func() int {
			virtualDeployments, err = f.VClusterClient.AppsV1().Deployments("default").List(f.Context, metav1.ListOptions{})
			framework.ExpectNoError(err)
			return len(virtualDeployments.Items)
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.Equal(1))

		// wait for pod to become ready
		var podList *corev1.PodList
		gomega.Eventually(func() bool {
			podList, err = f.VClusterClient.CoreV1().Pods("default").List(f.Context, metav1.ListOptions{})
			framework.ExpectNoError(err)
			return len(podList.Items) == 1 && podList.Items[0].Status.Phase == corev1.PodRunning
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// get pod in host cluster
		pod := &podList.Items[0]
		hostPod := &corev1.Pod{}
		err = f.HostCRClient.Get(f.Context, translate.Default.HostName(nil, pod.Name, pod.Namespace), hostPod)
		framework.ExpectNoError(err)

		// check if hook worked
		framework.ExpectEqual(hostPod.Labels["created-by-plugin"], "pod-hook")
	})

	ginkgo.It("check crd sync", func() {
		car := &examplev1.Car{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "kube-system",
			},
			Spec: examplev1.CarSpec{
				Type:  "Audi",
				Seats: 4,
			},
		}

		// create car in vcluster
		gomega.Eventually(func() bool {
			err := f.VClusterCRClient.Create(f.Context, car)
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// wait for car to become synced
		hostCar := &examplev1.Car{}
		carName := translate.Default.HostName(nil, car.Name, car.Namespace)
		gomega.Eventually(func() bool {
			err := f.HostCRClient.Get(f.Context, carName, hostCar)
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// check if car is synced correctly
		framework.ExpectEqual(car.Spec.Seats, hostCar.Spec.Seats)
		framework.ExpectEqual(car.Spec.Type, hostCar.Spec.Type)
	})

	ginkgo.It("check hooks work correctly", func() {
		// create a new service
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test123",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name: "test",
						Port: int32(1000),
					},
				},
			},
		}

		// create service
		err := f.VClusterCRClient.Create(f.Context, service)
		framework.ExpectNoError(err)

		// wait for service to become synced
		hostService := &corev1.Service{}
		gomega.Eventually(func() bool {
			err := f.HostCRClient.Get(f.Context, translate.Default.HostName(nil, service.Name, service.Namespace), hostService)
			return err == nil
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())

		// check if service is synced correctly
		framework.ExpectEqual(len(hostService.Spec.Ports), 2)
		framework.ExpectEqual(hostService.Spec.Ports[1].Name, "plugin")
		framework.ExpectEqual(hostService.Spec.Ports[1].Port, int32(19000))
	})

	ginkgo.It("check the interceptor", func() {
		// wait for secret to become synced
		vPod := &corev1.Pod{}
		err := f.VClusterCRClient.Get(f.Context, types.NamespacedName{Name: "stuff", Namespace: "test"}, vPod)
		framework.ExpectNoError(err)

		// check if secret is synced correctly
		framework.ExpectEqual(vPod.Name, "definitelynotstuff")
	})
})
