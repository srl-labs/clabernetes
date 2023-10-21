package topology

import (
	"context"
	"fmt"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) resolveFabricServices(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) (*clabernetescontrollers.ResolvedServices, error) {
	ownedServices := &k8scorev1.ServiceList{}

	err := r.Client.List(
		ctx,
		ownedServices,
		ctrlruntimeclient.InNamespace(obj.GetNamespace()),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: obj.GetName(),
		},
	)
	if err != nil {
		r.Log.Criticalf("failed fetching owned services, error: '%s'", err)

		return nil, err
	}

	services := &clabernetescontrollers.ResolvedServices{
		Current: map[string]*k8scorev1.Service{},
	}

	for i := range ownedServices.Items {
		labels := ownedServices.Items[i].Labels

		if labels == nil {
			return nil, fmt.Errorf(
				"%w: labels are nil, but we expect to see topology owner label here",
				claberneteserrors.ErrInvalidData,
			)
		}

		topologyServiceType := labels[clabernetesconstants.LabelTopologyServiceType]

		if topologyServiceType != clabernetesconstants.TopologyServiceTypeFabric {
			// not the kind of service we're looking for here, we only care about the services
			// used for connecting the nodes together here.
			continue
		}

		nodeName, ok := labels[clabernetesconstants.LabelTopologyNode]
		if !ok || nodeName == "" {
			return nil, fmt.Errorf(
				"%w: topology node label is missing or empty",
				claberneteserrors.ErrInvalidData,
			)
		}

		services.Current[nodeName] = &ownedServices.Items[i]
	}

	allNodes := make([]string, len(clabernetesConfigs))

	var nodeIdx int

	for nodeName := range clabernetesConfigs {
		allNodes[nodeIdx] = nodeName

		nodeIdx++
	}

	services.Missing = clabernetesutil.StringSliceDifference(
		services.CurrentServiceNames(),
		allNodes,
	)

	r.Log.Debugf(
		"services are missing for the following nodes: %s",
		services.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		allNodes,
		services.CurrentServiceNames(),
	)

	r.Log.Debugf(
		"extraneous services exist for following nodes: %s",
		extraEndpointDeployments,
	)

	services.Extra = make([]*k8scorev1.Service, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		services.Extra[idx] = services.Current[endpoint]
	}

	return services, nil
}

func (r *Reconciler) pruneFabricServices(
	ctx context.Context,
	services *clabernetescontrollers.ResolvedServices,
) error {
	r.Log.Info("pruning extraneous services")

	for _, extraDeployment := range services.Extra {
		r.Log.Debugf(
			"removing service '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := r.Client.Delete(ctx, extraDeployment)
		if err != nil {
			r.Log.Criticalf(
				"failed removing service '%s/%s' error: %s",
				extraDeployment.Namespace,
				extraDeployment.Name,
				err,
			)

			return err
		}
	}

	return nil
}

func (r *Reconciler) enforceFabricServices(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
	services *clabernetescontrollers.ResolvedServices,
) error {
	r.Log.Info("creating missing services")

	for _, nodeName := range services.Missing {
		service := r.renderFabricService(
			obj,
			nodeName,
		)

		err := ctrlruntimeutil.SetOwnerReference(obj, service, r.Client.Scheme())
		if err != nil {
			return err
		}

		r.Log.Debugf(
			"creating service '%s/%s'",
			service.Namespace,
			service.Name,
		)

		err = r.Client.Create(ctx, service)
		if err != nil {
			r.Log.Criticalf(
				"failed creating service '%s/%s' error: %s",
				service.Namespace,
				service.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	r.Log.Info("enforcing desired state on existing services")

	for nodeName, service := range services.Current {
		r.Log.Debugf(
			"comparing existing service '%s/%s' to desired state",
			service.Namespace,
			service.Name,
		)

		expectedService := r.renderFabricService(
			obj,
			nodeName,
		)

		err := ctrlruntimeutil.SetOwnerReference(obj, service, r.Client.Scheme())
		if err != nil {
			return err
		}

		if !ServiceConforms(service, expectedService, obj.GetUID()) {
			r.Log.Debugf(
				"comparing existing service '%s/%s' spec does not conform to desired state, "+
					"updating",
				service.Namespace,
				service.Name,
			)

			err = r.Client.Update(ctx, expectedService)
			if err != nil {
				r.Log.Criticalf(
					"failed updating service '%s/%s' error: %s",
					expectedService.Namespace,
					expectedService.Name,
					err,
				)

				return err
			}
		}
	}

	return nil
}

func (r *Reconciler) renderFabricService(
	obj ctrlruntimeclient.Object,
	nodeName string,
) *k8scorev1.Service {
	name := obj.GetName()

	serviceName := fmt.Sprintf("%s-%s-vx", name, nodeName)

	labels := map[string]string{
		clabernetesconstants.LabelApp:                 clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:                serviceName,
		clabernetesconstants.LabelTopologyOwner:       name,
		clabernetesconstants.LabelTopologyNode:        nodeName,
		clabernetesconstants.LabelTopologyKind:        r.ResourceKind,
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeFabric, //nolint:lll
	}

	service := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: obj.GetNamespace(),
			Labels:    labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Ports: []k8scorev1.ServicePort{
				{
					Name:     "vxlan",
					Protocol: "UDP",
					Port:     clabernetesconstants.VXLANServicePort,
					TargetPort: intstr.IntOrString{
						IntVal: clabernetesconstants.VXLANServicePort,
					},
				},
			},
			Selector: map[string]string{
				clabernetesconstants.LabelTopologyOwner: name,
				clabernetesconstants.LabelTopologyNode:  nodeName,
			},
			Type: "ClusterIP",
		},
	}

	return service
}
