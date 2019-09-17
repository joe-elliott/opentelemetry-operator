package opentelemetryservice

import (
	"testing"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

func TestProperServiceMonitor(t *testing.T) {
	// test
	s := serviceMonitor(ctx)
	backingSvc := monitoringService(ctx)

	// verify
	assert.Equal(t, "my-otelsvc-collector", s.Name)
	assert.Equal(t, "custom-annotation-value", s.Annotations["custom-annotation"])
	assert.Equal(t, "custom-value", s.Labels["custom-label"])
	assert.Equal(t, s.Name, s.Labels["app.kubernetes.io/name"])
	assert.Equal(t, backingSvc.Labels, s.Spec.Selector.MatchLabels)
}

func TestProperReconcileServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	schem := scheme.Scheme
	schem.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.OpenTelemetryService{})
	schem.AddKnownTypes(monitoringv1.SchemeGroupVersion, &monitoringv1.ServiceMonitor{}, &monitoringv1.ServiceMonitorList{})
	reconciler := New(cl, schem)

	// test
	req := reconcile.Request{}
	_, err := reconciler.Reconcile(req)
	assert.NoError(t, err)

	// verify
	list := &monitoringv1.ServiceMonitorList{}
	cl.List(ctx, client.InNamespace(instance.Namespace), list)

	// we assert the correctness of the service in another test
	assert.Len(t, list.Items, 1)

	// we assert the correctness of the reference in another test
	for _, item := range list.Items {
		assert.Len(t, item.OwnerReferences, 1)
	}
}

func TestUpdateServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	schem := scheme.Scheme
	schem.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.OpenTelemetryService{})
	schem.AddKnownTypes(monitoringv1.SchemeGroupVersion, &monitoringv1.ServiceMonitor{}, &monitoringv1.ServiceMonitorList{})

	c := serviceMonitor(ctx)
	c.Annotations = nil
	c.Labels = nil

	cl := fake.NewFakeClient(c)
	reconciler := New(cl, schem)

	// sanity check
	persisted := &monitoringv1.ServiceMonitor{}
	assert.NoError(t, cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))

	// test
	err := reconciler.reconcileServiceMonitor(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &monitoringv1.ServiceMonitor{}
	err = cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)
	assert.NoError(t, err)
}

func TestDeleteExtraServiceMonitor(t *testing.T) {
	// prepare
	viper.Set(opentelemetry.SvcMonitorAvailable, true)
	defer viper.Reset()

	schem := scheme.Scheme
	schem.AddKnownTypes(v1alpha1.SchemeGroupVersion, &v1alpha1.OpenTelemetryService{})
	schem.AddKnownTypes(monitoringv1.SchemeGroupVersion, &monitoringv1.ServiceMonitor{}, &monitoringv1.ServiceMonitorList{})

	c := serviceMonitor(ctx)
	c.Name = "extra-service"

	cl := fake.NewFakeClient(c)
	reconciler := New(cl, schem)

	// sanity check
	persisted := &monitoringv1.ServiceMonitor{}
	assert.NoError(t, cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted))

	// test
	err := reconciler.reconcileServiceMonitor(ctx)
	assert.NoError(t, err)

	// verify
	persisted = &monitoringv1.ServiceMonitor{}
	err = cl.Get(ctx, types.NamespacedName{Name: c.Name, Namespace: c.Namespace}, persisted)

	assert.Empty(t, persisted.Name)
	assert.Error(t, err) // not found
}
