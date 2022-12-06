package deploymentmetadata

import (
	"context"
	"fmt"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	orchestrationTech = "Operator"
	argumentPrefix    = `--set-deployment-metadata=`

	keyOperatorScriptVersion = "script_version"
	keyOrchestratorID        = "orchestrator_id"
	keyOrchestrationTech     = "orchestration_tech"
)

var (
	// Iterating over a map is not consistent,
	// and we have to keep the order always the same so we don't restart the pods unnecessarily
	orderedKeys = []string{
		keyOperatorScriptVersion,
		keyOrchestrationTech,
		keyOrchestratorID,
	}
)

type DeploymentMetadata struct {
	OrchestratorID string
	DeploymentType string
}

func NewDeploymentMetadata(orchestratorID string, dt string) *DeploymentMetadata {
	return &DeploymentMetadata{OrchestratorID: orchestratorID, DeploymentType: dt}
}

func (metadata DeploymentMetadata) StoreInConfigMap(ctx context.Context, kubeClient client.Client, dynakube dynatracev1beta1.DynaKube) error {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatMetadataConfigMapName(dynakube.Name),
			Namespace: dynakube.Namespace,
		},
		Data: metadata.AsArgumentMap(),
	}
	err := kubeClient.Update(ctx, &configMap)
	if k8serrors.IsNotFound(err) {
		return kubeClient.Create(ctx, &configMap)
	}
	return err
}

func (metadata DeploymentMetadata) AsEnvsFromConfigMap(dkName string) []corev1.EnvVar {
	envs := []corev1.EnvVar{}
	for _, key := range orderedKeys {
		envs = append(envs, corev1.EnvVar{
			Name: strings.ToUpper(key),
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: formatMetadataConfigMapName(dkName),
					},
					Key: key,
				},
			},
		})
	}
	return envs
}

func (metadata DeploymentMetadata) AsArgsFromEnvs(dkName string) []string {
	return functional.Map(orderedKeys, formatMetadataArgFromEnv)
}

func (metadata DeploymentMetadata) AsArgumentMap() map[string]string {
	return map[string]string{
		keyOrchestrationTech:     formatMetadataArgument(keyOrchestrationTech, metadata.OrchestrationTech()),
		keyOperatorScriptVersion: formatMetadataArgument(keyOperatorScriptVersion, version.Version),
		keyOrchestratorID:        formatMetadataArgument(keyOrchestratorID, metadata.OrchestratorID),
	}
}

func (metadata DeploymentMetadata) AsString() string {
	res := []string{
		formatKeyValue(keyOrchestrationTech, metadata.OrchestrationTech()),
		formatKeyValue(keyOperatorScriptVersion, version.Version),
		formatKeyValue(keyOrchestratorID, metadata.OrchestratorID),
	}

	return strings.Join(res, ";")
}

func (metadata DeploymentMetadata) OrchestrationTech() string {
	return fmt.Sprintf("%s-%s", orchestrationTech, metadata.DeploymentType)
}

func formatKeyValue(key string, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func formatMetadataArgument(key string, value string) string {
	return fmt.Sprintf(`%s%s=%s`, argumentPrefix, key, value)
}

func formatMetadataArgFromEnv(key string) string {
	out := strings.ToUpper(key)
	return fmt.Sprintf("\"$(%s)\"", out)
}

func formatMetadataConfigMapName(key string) string {
	return fmt.Sprintf(`%s-deployment-metadata`, key)
}
