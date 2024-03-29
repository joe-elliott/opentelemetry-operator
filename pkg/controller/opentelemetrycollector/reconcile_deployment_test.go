package opentelemetrycollector

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

func TestProperDeployment(t *testing.T) {
	// test
	d := deployment(ctx)

	// verify
	assert.Equal(t, d.Name, "my-otelcol-collector")
	assert.Equal(t, d.Annotations["custom-annotation"], "custom-annotation-value")
	assert.Equal(t, d.Labels["custom-label"], "custom-value")
	assert.Equal(t, d.Labels["app.kubernetes.io/name"], d.Name)
}

func TestProperDeployments(t *testing.T) {
	// test
	d := deployments(ctx)

	// verify
	assert.Len(t, d, 1)
}

func TestNoDeploymentsWhenModeDaemonSet(t *testing.T) {
	// test
	instance := &v1alpha1.OpenTelemetryCollector{}
	instance.Spec.Mode = opentelemetry.ModeDaemonSet
	ctx := context.WithValue(ctx, opentelemetry.ContextInstance, instance)

	// prepare
	d := deployments(ctx)

	// verify
	assert.Len(t, d, 0)
}

func TestDeploymentOverridesConfig(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{"config": "custom-path"},
		},
	}
	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, instance)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	// test
	d := deployment(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Len(t, d.Spec.Template.Spec.Containers[0].Args, 1)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Args[0], "/conf/collector.yaml")
}

func TestProperReconcileDeployment(t *testing.T) {
	// prepare
	clients := &Clients{
		client: fake.NewFakeClient(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{}

	// test
	reconciler.Reconcile(req)

	// verify
	list := &appsv1.DeploymentList{}
	clients.client.List(ctx, list, client.InNamespace(instance.Namespace))

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	assert.Len(t, list.Items[0].OwnerReferences, 1)
}

func TestOverrideDeploymentImageFromCustomResource(t *testing.T) {
	// prepare
	instance := &v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "myrepo/custom-image:version",
		},
	}
	ctx := context.WithValue(ctx, opentelemetry.ContextInstance, instance)

	// test
	d := deployment(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "myrepo/custom-image:version", d.Spec.Template.Spec.Containers[0].Image)
}

func TestOverrideDeploymentImageFromCLI(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.OtelColImageConfigKey, "myrepo/custom-image-cli:version")
	defer viper.Reset()
	defer opentelemetry.ResetFlagSet()

	// test
	d := deployment(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "myrepo/custom-image-cli:version", d.Spec.Template.Spec.Containers[0].Image)
}

func TestDefaultDeploymentImage(t *testing.T) {
	// prepare
	opentelemetry.FlagSet()
	defer opentelemetry.ResetFlagSet()

	// test
	d := deployment(ctx)

	// verify
	assert.Len(t, d.Spec.Template.Spec.Containers, 1)
	assert.Contains(t, d.Spec.Template.Spec.Containers[0].Image, "quay.io/opentelemetry/opentelemetry-collector")
}

func TestUpdateDeployment(t *testing.T) {
	// prepare
	clients := &Clients{
		client: fake.NewFakeClient(instance),
	}
	reconciler := New(schem, clients)
	req := reconcile.Request{}
	reconciler.Reconcile(req)

	// sanity check
	name := fmt.Sprintf("%s-collector", instance.Name)
	persisted := &appsv1.Deployment{}
	err := clients.client.Get(ctx, types.NamespacedName{Name: name, Namespace: instance.Namespace}, persisted)
	assert.NoError(t, err)

	// prepare the test object
	updated := *instance
	updated.Spec.Image = "custom-image"

	ctx := context.WithValue(context.Background(), opentelemetry.ContextInstance, &updated)
	ctx = context.WithValue(ctx, opentelemetry.ContextLogger, logf.Log.WithName("unit-tests"))

	// test
	reconciler.reconcileDeployment(ctx)

	// verify
	persisted = &appsv1.Deployment{}
	assert.NoError(t, clients.client.Get(ctx, types.NamespacedName{Name: name, Namespace: updated.Namespace}, persisted))
	assert.Len(t, persisted.Spec.Template.Spec.Containers, 1)
	assert.Equal(t, "custom-image", persisted.Spec.Template.Spec.Containers[0].Image)
}

func TestDeleteExtraDeployment(t *testing.T) {
	// prepare
	c := deployment(ctx)
	c.Name = "extra-deployment"

	clients := &Clients{
		client: fake.NewFakeClient(c),
	}
	reconciler := New(schem, clients)

	// sanity check
	persisted := &appsv1.Deployment{}
	assert.NoError(t, clients.client.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))

	// test
	err := reconciler.reconcileDeployment(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &appsv1.Deployment{}
	err = clients.client.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)

	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
