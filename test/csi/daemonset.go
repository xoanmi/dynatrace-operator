package csi

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	Name      = "dynatrace-oneagent-csi-driver"
	Namespace = operator.Namespace
)

func Get(ctx context.Context, resource *resources.Resources) (appsv1.DaemonSet, error) {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      Name,
		Namespace: Namespace,
	}).Get()
}

func ForEachPod(ctx context.Context, resource *resources.Resources, consumer daemonset.PodConsumer) error {
	return daemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      Name,
		Namespace: Namespace,
	}).ForEachPod(consumer)
}

func WaitForFileCleanup() func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
	return func(ctx context.Context, environmentConfig *envconf.Config, t *testing.T) (context.Context, error) {
		client, _ := environmentConfig.NewClient()

		pods := &corev1.PodList{}
		err := client.Resources(Namespace).List(context.TODO(), pods)
		if err != nil || pods.Items == nil {
			t.Error("error while getting pods", err)
		}

		csiDriverPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
			return strings.Contains(podItem.Name, Name)
		})
		require.NotEmpty(t, csiDriverPods)

		for _, csiPod := range csiDriverPods {
			_, err := pod.ExecuteNg(ctx, client, csiPod, "server", "rm", "-rf", "/data/*")
			assert.NoError(t, err)
			return ctx, nil
		}

		return ctx, nil
	}
}
