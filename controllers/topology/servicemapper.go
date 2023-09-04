package topology

import (
	"context"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	k8scorev1 "k8s.io/api/core/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimereconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MapServiceToContainerlab is a generic func that topology controllers can use to map services
// back to clabernetes objects. We watch services so we can update the lb address in the topology
// objects status.
func (r *Reconciler) MapServiceToContainerlab(
	_ context.Context,
	obj ctrlruntimeclient.Object,
) []ctrlruntimereconcile.Request {
	service, ok := obj.(*k8scorev1.Service)
	if !ok {
		r.Log.Critical(
			"failed casting object to service in service to containerlab map func, " +
				"this should not happen. continuing but will not schedule any reconciles for this" +
				"service object",
		)

		return nil
	}

	labels := service.GetLabels()

	_, clabernetesOk := labels[clabernetesconstants.LabelApp]
	if !clabernetesOk {
		r.Log.Debugf(
			"service '%s/%s' is not a clabernetes service, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	clabernetesTopologyKind, clabernetesKindOk := labels[clabernetesconstants.LabelTopologyKind]
	if !clabernetesKindOk {
		r.Log.Debugf(
			"service '%s/%s' is not a clabernetes service, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	if clabernetesTopologyKind != r.ResourceKind {
		r.Log.Debugf(
			"service '%s/%s' is not a service for this controllers resource kind, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	_, clabernetesExposeOk := labels[clabernetesconstants.LabelTopologyServiceType]
	if !clabernetesExposeOk {
		r.Log.Debugf(
			"service '%s/%s' is not a clabernetes expose service, ignoring",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	clabResource, clabResourceOk := labels[clabernetesconstants.LabelTopologyOwner]
	if !clabResourceOk {
		r.Log.Criticalf(
			"service '%s/%s' is a clabernetes expose service, but cannot determine"+
				" corresponding clabernetes resource",
			service.Namespace,
			service.Name,
		)

		return nil
	}

	r.Log.Infof(
		"service '%s/%s' is clabernetes expose service,"+
			" scheduling reconcile for containerlab resource '%s/%s' ",
		service.Namespace,
		service.Name,
		service.Namespace,
		clabResource,
	)

	return []ctrlruntimereconcile.Request{
		{
			NamespacedName: apimachinerytypes.NamespacedName{
				Namespace: service.Namespace,
				Name:      clabResource,
			},
		},
	}
}
