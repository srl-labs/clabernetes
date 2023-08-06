package containerlab

import (
	"context"
	"fmt"

	clabernetescontainerlab "gitlab.com/carlmontanari/clabernetes/containerlab"

	"k8s.io/apimachinery/pkg/util/intstr"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Controller) reconcileServices(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*clabernetescontainerlab.Config,
) error {
	services, err := c.resolveServices(ctx, clab, clabernetesConfigs)
	if err != nil {
		return err
	}

	err = c.pruneServices(ctx, services)
	if err != nil {
		return err
	}

	err = c.enforceServices(ctx, clab, services)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) resolveServices(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*clabernetescontainerlab.Config,
) (*clabernetescontrollers.ResolvedServices, error) {
	ownedServices := &k8scorev1.ServiceList{}

	err := c.Client.List(
		ctx,
		ownedServices,
		ctrlruntimeclient.InNamespace(clab.Namespace),
		ctrlruntimeclient.MatchingLabels{
			clabernetesconstants.LabelTopologyOwner: clab.Name,
		},
	)
	if err != nil {
		c.Log.Criticalf("failed fetching owned services, error: '%s'", err)

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

	c.BaseController.Log.Debugf(
		"services are missing for the following nodes: %s",
		services.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		allNodes,
		services.CurrentServiceNames(),
	)

	c.BaseController.Log.Debugf(
		"extraneous services exist for following nodes: %s",
		extraEndpointDeployments,
	)

	services.Extra = make([]*k8scorev1.Service, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		services.Extra[idx] = services.Current[endpoint]
	}

	return services, nil
}

func (c *Controller) pruneServices(
	ctx context.Context,
	services *clabernetescontrollers.ResolvedServices,
) error {
	c.BaseController.Log.Info("pruning extraneous services")

	for _, extraDeployment := range services.Extra {
		c.BaseController.Log.Debugf(
			"removing service '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := c.Client.Delete(ctx, extraDeployment)
		if err != nil {
			c.BaseController.Log.Criticalf(
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

func (c *Controller) enforceServices( //nolint:dupl
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	services *clabernetescontrollers.ResolvedServices,
) error {
	c.BaseController.Log.Info("creating missing services")

	for _, nodeName := range services.Missing {
		service := renderService(
			clab,
			nodeName,
		)

		err := c.enforceServiceOwnerReference(clab, service)
		if err != nil {
			return err
		}

		c.BaseController.Log.Debugf(
			"creating service '%s/%s'",
			service.Namespace,
			service.Name,
		)

		err = c.Client.Create(ctx, service)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed creating service '%s/%s' error: %s",
				service.Namespace,
				service.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	c.BaseController.Log.Info("enforcing desired state on existing services")

	for nodeName, service := range services.Current {
		c.BaseController.Log.Debugf(
			"comparing existing service '%s/%s' to desired state",
			service.Namespace,
			service.Name,
		)

		expectedService := renderService(
			clab,
			nodeName,
		)

		err := c.enforceServiceOwnerReference(clab, expectedService)
		if err != nil {
			return err
		}

		if !serviceConforms(service, expectedService, clab.UID) {
			c.BaseController.Log.Debugf(
				"comparing existing service '%s/%s' spec does not conform to desired state, "+
					"updating",
				service.Namespace,
				service.Name,
			)

			err = c.Client.Update(ctx, expectedService)
			if err != nil {
				c.BaseController.Log.Criticalf(
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

func renderService(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	nodeName string,
) *k8scorev1.Service {
	serviceName := fmt.Sprintf("%s-%s", clab.Name, nodeName)

	labels := map[string]string{
		clabernetesconstants.LabelApp:                 clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:                serviceName,
		clabernetesconstants.LabelTopologyOwner:       clab.Name,
		clabernetesconstants.LabelTopologyNode:        nodeName,
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeFabric, //nolint:lll
	}

	service := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: clab.Namespace,
			Labels:    labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Ports: []k8scorev1.ServicePort{
				{
					Name:     "vxlan",
					Protocol: "UDP",
					Port:     clabernetesconstants.VXLANPort,
					TargetPort: intstr.IntOrString{
						IntVal: clabernetesconstants.VXLANPort,
					},
				},
			},
			Selector: map[string]string{
				clabernetesconstants.LabelTopologyOwner: clab.Name,
				clabernetesconstants.LabelTopologyNode:  nodeName,
			},
			Type: "ClusterIP",
		},
	}

	return service
}
