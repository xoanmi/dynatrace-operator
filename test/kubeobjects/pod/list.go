package pod

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

func List(t *testing.T, ctx context.Context, resource *resources.Resources, namespaceName string) corev1.PodList {
	var pods corev1.PodList
	require.NoError(t, resource.WithNamespace(namespaceName).List(ctx, &pods))
	return pods
}

func ListFilteredByName(t *testing.T, ctx context.Context, resource *resources.Resources, namespaceName string, filter string) []corev1.Pod {

	pods := List(t, ctx, resource, namespaceName)

	filteredPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		return strings.Contains(podItem.Name, filter)
	})

	return filteredPods
}
