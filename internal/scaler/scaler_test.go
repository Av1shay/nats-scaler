package scaler

import (
	"context"
	"testing"

	internalTypes "github.com/Av1shay/nats-scaler/internal/types"
	testutils "github.com/Av1shay/nats-scaler/test/utils"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRealScaler_ReconcileScale(t *testing.T) {
	fakeLogger := &testutils.FakeLogger{}

	ctx := logr.NewContext(context.Background(), logr.New(fakeLogger))
	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	initialReplicas := int32(1)

	const (
		deploymentName = "my-deploy"
		namespace      = "default"
	)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
		},
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(deploy).Build()

	scaler := NewScaler(WithCooldown(0))
	nn := types.NamespacedName{
		Name: deploymentName, Namespace: namespace,
	}

	t.Run("scale up", func(t *testing.T) {
		err := scaler.ReconcileScale(ctx, k8sClient, nn, internalTypes.ScalerParams{
			ScaleUpThreshold:   10,
			ScaleDownThreshold: 3,
			MinReplicas:        1,
			MaxReplicas:        5,
		}, 20)
		require.NoError(t, err)

		var updated appsv1.Deployment
		err = k8sClient.Get(ctx, nn, &updated)
		require.NoError(t, err)

		require.Equal(t, int32(2), *updated.Spec.Replicas)

		// check logs
		log := fakeLogger.Buff.String()
		require.Equal(t, "Scaling up: 1 → 2 (pending: 20 > 10)", log)
		fakeLogger.Buff.Reset()
	})

	t.Run("scale down", func(t *testing.T) {
		err := scaler.ReconcileScale(ctx, k8sClient, nn, internalTypes.ScalerParams{
			ScaleUpThreshold:   10,
			ScaleDownThreshold: 3,
			MinReplicas:        1,
			MaxReplicas:        5,
		}, 2)
		require.NoError(t, err)

		var updated appsv1.Deployment
		err = k8sClient.Get(ctx, nn, &updated)
		require.NoError(t, err)

		require.Equal(t, int32(1), *updated.Spec.Replicas)

		log := fakeLogger.Buff.String()
		require.Equal(t, "Scaling down: 2 → 1 (pending: 2 < 3)", log)
	})

	_, ok := lastScaleMap.Load(nn)
	require.True(t, ok)
}
