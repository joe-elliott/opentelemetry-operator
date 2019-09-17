package opentelemetryservice

import (
	"context"
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// reconcileServiceMonitor reconciles the service monitor(s) required for the instance in the current context
func (r *ReconcileOpenTelemetryService) reconcileServiceMonitor(ctx context.Context) error {
	if !viper.GetBool(opentelemetry.SvcMonitorAvailable) {
		logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
		logger.V(2).Info("skipping reconciliation for service monitor, as the CRD isn't registered with the cluster")
		return nil
	}

	svcs := []*monitoringv1.ServiceMonitor{
		serviceMonitor(ctx),
	}

	// first, handle the create/update parts
	if err := r.reconcileExpectedServiceMonitors(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the expected service monitors: %v", err)
	}

	// then, delete the extra objects
	if err := r.deleteServiceMonitors(ctx, svcs); err != nil {
		return fmt.Errorf("failed to reconcile the service monitors to be deleted: %v", err)
	}

	return nil
}

func serviceMonitor(ctx context.Context) *monitoringv1.ServiceMonitor {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryService)
	name := fmt.Sprintf("%s-collector", instance.Name)

	labels := commonLabels(ctx)
	labels["app.kubernetes.io/name"] = name

	selector := commonLabels(ctx)
	selector["app.kubernetes.io/name"] = fmt.Sprintf("%s-monitoring", name)

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: selector,
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port: "monitoring",
			}},
		},
	}

}

func (r *ReconcileOpenTelemetryService) reconcileExpectedServiceMonitors(ctx context.Context, expected []*monitoringv1.ServiceMonitor) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	for _, obj := range expected {
		desired := obj
		r.setControllerReference(ctx, desired)

		existing := &monitoringv1.ServiceMonitor{}
		err := r.client.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
		if err != nil && errors.IsNotFound(err) {
			if err := r.client.Create(ctx, desired); err != nil {
				return fmt.Errorf("failed to create: %v", err)
			}

			logger.WithValues("svcmon.name", desired.Name, "svcmon.namespace", desired.Namespace).V(2).Info("created")
			continue
		} else if err != nil {
			return fmt.Errorf("failed to retrieve: %v", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if err := r.client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes to service monitor: %v", err)
		}
		logger.V(2).Info("applied", "svcmon.name", desired.Name, "svcmon.namespace", desired.Namespace)
	}

	return nil
}

func (r *ReconcileOpenTelemetryService) deleteServiceMonitors(ctx context.Context, expected []*monitoringv1.ServiceMonitor) error {
	instance := ctx.Value(opentelemetry.ContextInstance).(*v1alpha1.OpenTelemetryService)
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	opts := client.InNamespace(instance.Namespace).MatchingLabels(map[string]string{
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", instance.Namespace, instance.Name),
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	})
	list := &monitoringv1.ServiceMonitorList{}
	if err := r.client.List(ctx, opts, list); err != nil {
		return fmt.Errorf("failed to list: %v", err)
	}

	for _, existing := range list.Items {
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := r.client.Delete(ctx, existing); err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
			logger.V(2).Info("deleted", "svcmon.name", existing.Name, "svcmon.namespace", existing.Namespace)
		}
	}

	return nil
}
