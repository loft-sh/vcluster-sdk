package syncers

import (
	"net/http"

	"github.com/loft-sh/vcluster-sdk/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	corev1 "k8s.io/api/core/v1"
)

var _ plugin.Interceptor = DummyInterceptor{}

type DummyInterceptor struct{}

func (d DummyInterceptor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)

	s := serializer.NewCodecFactory(scheme)
	responsewriters.WriteObjectNegotiated(
		s,
		negotiation.DefaultEndpointRestrictions,
		schema.GroupVersion{
			Group:   "",
			Version: "v1",
		},
		w,
		r,
		200,
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "definitelynotstuff"}},
		false)
}

func (d DummyInterceptor) Name() string {
	return "testinterceptor"
}

func (d DummyInterceptor) InterceptionRules() []v2.InterceptorRule {
	return []v2.InterceptorRule{
		{
			APIGroups:     []string{"*"},
			Resources:     []string{"pods"},
			ResourceNames: []string{"*"},
			Verbs:         []string{"get"},
		},
	}
}
