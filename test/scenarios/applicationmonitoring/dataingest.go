//go:build e2e

package applicationmonitoring

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	metadataFile = "/var/lib/dynatrace/enrichment/dt_metadata.json"
)

type metadata struct {
	WorkloadKind string `json:"dt.kubernetes.workload.kind,omitempty"`
	WorkloadName string `json:"dt.kubernetes.workload.name,omitempty"`
}

func dataIngest(t *testing.T) features.Feature {
	builder := features.New("data-ingest")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(false),
		}).Build()

	sampleDeployment := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleDeployment.WithAnnotations(map[string]string{
		webhook.AnnotationOneAgentInject:   "false",
		webhook.AnnotationDataIngestInject: "true",
	})
	samplePod := sampleapps.NewSamplePod(t, testDynakube)
	samplePod.WithAnnotations(map[string]string{
		webhook.AnnotationOneAgentInject:   "false",
		webhook.AnnotationDataIngestInject: "true",
	})

	// Register operator + dynakube install
	assess.InstallDynatrace(builder, &secretConfig, testDynakube)

	// Register actual test (+sample cleanup)
	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("install sample pod  and wait till ready", samplePod.Install())
	builder.Assess("deployment pods only have data ingest", deploymentPodsHaveOnlyDataIngestInitContainer(sampleDeployment))
	builder.Assess("pod only has data ingest", podHasOnlyDataIngestInitContainer(samplePod))

	builder.WithTeardown("removing samples", sampleDeployment.UninstallNamespace())

	// Register operator + dynakube uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func podHasOnlyDataIngestInitContainer(samplePod sampleapps.SampleApp) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		testPod := samplePod.Get(ctx, t, environmentConfig.Client().Resources()).(*corev1.Pod)

		assessOnlyDataIngestIsInjected(t)(*testPod)
		assessPodHasDataIngestFile(t, environmentConfig.Client().RESTConfig(), *testPod)

		return ctx
	}
}

func assessPodHasDataIngestFile(t *testing.T, restConfig *rest.Config, testPod corev1.Pod) {
	dataIngestMetadata := getDataIngestMetadataFromPod(t, restConfig, testPod)

	assert.Equal(t, dataIngestMetadata.WorkloadKind, "Pod")
	assert.Equal(t, dataIngestMetadata.WorkloadName, testPod.Name)
}

func deploymentPodsHaveOnlyDataIngestInitContainer(sampleApp sampleapps.SampleApp) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		query := deployment.NewQuery(ctx, environmentConfig.Client().Resources(), client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		})
		err := query.ForEachPod(assessOnlyDataIngestIsInjected(t))

		require.NoError(t, err)

		err = query.ForEachPod(assessDeploymentHasDataIngestFile(t, environmentConfig.Client().RESTConfig(), sampleApp.Name()))

		require.NoError(t, err)

		return ctx
	}
}

func assessDeploymentHasDataIngestFile(t *testing.T, restConfig *rest.Config, deploymentName string) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		dataIngestMetadata := getDataIngestMetadataFromPod(t, restConfig, pod)

		assert.Equal(t, dataIngestMetadata.WorkloadKind, "Deployment")
		assert.Equal(t, dataIngestMetadata.WorkloadName, deploymentName)
	}
}

func getDataIngestMetadataFromPod(t *testing.T, restConfig *rest.Config, dataIngestPod corev1.Pod) metadata {
	query := pod.NewExecutionQuery(dataIngestPod, dataIngestPod.Spec.Containers[0].Name, shell.ReadFile(metadataFile)...)
	result, err := query.Execute(restConfig)

	require.NoError(t, err)

	assert.Zero(t, result.StdErr.Len())
	assert.NotEmpty(t, result.StdOut)

	var dataIngestMetadata metadata
	err = json.Unmarshal(result.StdOut.Bytes(), &dataIngestMetadata)

	require.NoError(t, err)

	return dataIngestMetadata
}

func assessOnlyDataIngestIsInjected(t *testing.T) deployment.PodConsumer {
	return func(pod corev1.Pod) {
		initContainers := pod.Spec.InitContainers

		require.Len(t, initContainers, 1)

		installOneAgentContainer := initContainers[0]
		envVars := installOneAgentContainer.Env

		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentWorkloadKindEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentWorkloadNameEnv))
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, config.EnrichmentInjectedEnv))

		assert.False(t, kubeobjects.EnvVarIsIn(envVars, config.AgentInjectedEnv))
	}
}
