package plugin

import (
	"time"

	"github.com/loft-sh/vcluster/test/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			virtualDeployments, err = f.VclusterClient.AppsV1().Deployments("default").List(f.Context, metav1.ListOptions{})
			framework.ExpectNoError(err)
			return len(virtualDeployments.Items)
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.Equal(1))

		// wait for pod to become ready
		gomega.Eventually(func() bool {
			podList, err := f.VclusterClient.CoreV1().Pods("default").List(f.Context, metav1.ListOptions{})
			framework.ExpectNoError(err)
			return len(podList.Items) == 1 && podList.Items[0].Status.Phase == corev1.PodRunning
		}).
			WithPolling(pollingInterval).
			WithTimeout(pollingDurationLong).
			Should(gomega.BeTrue())
	})
})
