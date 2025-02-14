//go:build e2e

package istio

import (
	"context"
	"os"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	istioclientset "istio.io/client-go/pkg/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	istioNamespace         = "istio-system"
	istioInitContainerName = "istio-init"
	enforceIstioEnv        = "ENFORCE_ISTIO"
)

var IstioLabel = map[string]string{
	"istio-injection": "enabled",
}

func enforceIstio() bool {
	return os.Getenv(enforceIstioEnv) == "true"
}

func AssertIstioNamespace() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var namespace corev1.Namespace
		err := environmentConfig.Client().Resources().Get(ctx, istioNamespace, "", &namespace)
		if err != nil && !enforceIstio() {
			t.Skip("skipping istio test, istio namespace is not present")
			return ctx, nil
		}
		return ctx, errors.WithStack(err)
	}
}

func AssertIstiodDeployment() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		var deployment appsv1.Deployment
		err := environmentConfig.Client().Resources().Get(ctx, "istiod", "istio-system", &deployment)
		if err != nil && !enforceIstio() {
			t.Skip("skipping istio test, istiod deployment is not present")
			return ctx, nil
		}
		return ctx, errors.WithStack(err)
	}
}

func AssessIstio(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube, sampleApp sampleapps.SampleApp) {
	builder.Assess("sample apps have working istio init container", checkSampleAppIstioInitContainers(sampleApp, testDynakube))
	builder.Assess("operator pods have working istio init container", checkOperatorIstioInitContainers(testDynakube))
	builder.Assess("istio virtual service for ApiUrl created", checkVirtualServiceForApiUrl(testDynakube))
	builder.Assess("istio service entry for ApiUrl created", checkServiceEntryForApiUrl(testDynakube))
}

func checkSampleAppIstioInitContainers(sampleApp sampleapps.SampleApp, testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)
		assertIstioInitContainer(t, pods, testDynakube)
		return ctx
	}
}

func checkOperatorIstioInitContainers(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var pods corev1.PodList
		require.NoError(t, resources.WithNamespace(testDynakube.Namespace).List(ctx, &pods))

		assertIstioInitContainer(t, pods, testDynakube)
		return ctx
	}
}

func assertIstioInitContainer(t *testing.T, pods corev1.PodList, testDynakube dynatracev1beta1.DynaKube) {
	for _, podItem := range pods.Items {
		if podItem.DeletionTimestamp != nil {
			continue
		}

		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)

		if strings.HasPrefix(podItem.Name, testDynakube.OneAgentDaemonsetName()) {
			continue
		}

		require.NotEmpty(t, podItem.Spec.InitContainers, "'%s' pod has no init containers", podItem.Name)

		istioInitFound := false

		for _, initContainer := range podItem.Spec.InitContainers {
			if initContainer.Name == istioInitContainerName {
				istioInitFound = true
				break
			}
		}
		assert.True(t, istioInitFound, "'%s' pod - '%s' init container not found", podItem.Name, istioInitContainerName)
	}
}

func checkVirtualServiceForApiUrl(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		apiHost := apiUrlCommunicationHost(t)
		serviceName := istio.BuildNameForEndpoint(dynakube.Name, apiHost.Protocol, apiHost.Host, apiHost.Port)

		virtualService, err := istioClient(t, environmentConfig.Client().RESTConfig()).NetworkingV1alpha3().VirtualServices(dynakube.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.Nil(t, err, "istio: faild to get '%s' virtual service object", serviceName)

		require.NotEmpty(t, virtualService.ObjectMeta.OwnerReferences)
		assert.Equal(t, dynakube.Name, virtualService.ObjectMeta.OwnerReferences[0].Name)

		require.NotEmpty(t, virtualService.Spec.Hosts)
		assert.Equal(t, apiHost.Host, virtualService.Spec.Hosts[0])

		return ctx
	}
}

func checkServiceEntryForApiUrl(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		apiHost := apiUrlCommunicationHost(t)
		serviceName := istio.BuildNameForEndpoint(dynakube.Name, apiHost.Protocol, apiHost.Host, apiHost.Port)

		serviceEntry, err := istioClient(t, environmentConfig.Client().RESTConfig()).NetworkingV1alpha3().ServiceEntries(dynakube.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		require.Nil(t, err, "istio: failed to get '%s' service entry object", serviceName)

		require.NotEmpty(t, serviceEntry.ObjectMeta.OwnerReferences)
		assert.Equal(t, dynakube.Name, serviceEntry.ObjectMeta.OwnerReferences[0].Name)

		require.NotEmpty(t, serviceEntry.Spec.Hosts)
		assert.Equal(t, apiHost.Host, serviceEntry.Spec.Hosts[0])

		return ctx
	}
}

func istioClient(t *testing.T, restConfig *rest.Config) *istioclientset.Clientset {
	client, err := istioclientset.NewForConfig(restConfig)
	require.Nil(t, err, "istio: failed to initialize client")
	return client
}

func apiUrlCommunicationHost(t *testing.T) dtclient.CommunicationHost {
	secretConfig := tenant.GetSingleTenantSecret(t)

	apiHost, err := dtclient.ParseEndpoint(secretConfig.ApiUrl)
	require.Nil(t, err)

	return apiHost
}
