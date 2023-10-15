package topology

import (
	"context"
	"fmt"
	"sort"
	"strings"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
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

func (r *Reconciler) resolveExposeServices(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
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
		r.Log.Criticalf("failed fetching owned expose services, error: '%s'", err)

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

		if topologyServiceType != clabernetesconstants.TopologyServiceTypeExpose {
			// not the kind of service we're looking for here, we only care about the services
			// used for exposing nodes here.
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

	commonTopologySpec := obj.GetTopologyCommonSpec()

	exposedNodes := make([]string, 0)

	for nodeName, nodeData := range clabernetesConfigs {
		// if disable auto expose is true *and* there are no ports defined for the node *and*
		// there are no default ports defined for the topology we can skip the node from an expose
		// perspective.
		if commonTopologySpec.DisableAutoExpose &&
			len(nodeData.Topology.Nodes[nodeName].Ports) == 0 &&
			len(nodeData.Topology.Defaults.Ports) == 0 {
			continue
		}

		exposedNodes = append(exposedNodes, nodeName)
	}

	services.Missing = clabernetesutil.StringSliceDifference(
		services.CurrentServiceNames(),
		exposedNodes,
	)

	r.Log.Debugf(
		"expose services are missing for the following nodes: %s",
		services.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		exposedNodes,
		services.CurrentServiceNames(),
	)

	r.Log.Debugf(
		"extraneous expose services exist for following nodes: %s",
		extraEndpointDeployments,
	)

	services.Extra = make([]*k8scorev1.Service, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		services.Extra[idx] = services.Current[endpoint]
	}

	return services, nil
}

func (r *Reconciler) pruneExposeServices(
	ctx context.Context,
	services *clabernetescontrollers.ResolvedServices,
) error {
	r.Log.Info("pruning extraneous expose services")

	for _, extraDeployment := range services.Extra {
		r.Log.Debugf(
			"removing expose service '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := r.Client.Delete(ctx, extraDeployment)
		if err != nil {
			r.Log.Criticalf(
				"failed removing expose service '%s/%s' error: %s",
				extraDeployment.Namespace,
				extraDeployment.Name,
				err,
			)

			return err
		}
	}

	return nil
}

func (r *Reconciler) enforceExposeServices(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	objTopologyStatus clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	services *clabernetescontrollers.ResolvedServices,
) error {
	r.Log.Info("creating missing expose services")

	for _, nodeName := range services.Missing {
		service := r.renderExposeService(
			obj,
			objTopologyStatus,
			clabernetesConfigs,
			nodeName,
		)

		err := ctrlruntimeutil.SetOwnerReference(obj, service, r.Client.Scheme())
		if err != nil {
			return err
		}

		r.Log.Debugf(
			"creating expose service '%s/%s'",
			service.Namespace,
			service.Name,
		)

		err = r.Client.Create(ctx, service)
		if err != nil {
			r.Log.Criticalf(
				"failed creating expose service '%s/%s' error: %s",
				service.Namespace,
				service.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	r.Log.Info("enforcing desired state on existing expose services")

	for nodeName, service := range services.Current {
		r.Log.Debugf(
			"comparing existing expose service '%s/%s' to desired state",
			service.Namespace,
			service.Name,
		)

		expectedService := r.renderExposeService(
			obj,
			objTopologyStatus,
			clabernetesConfigs,
			nodeName,
		)

		if len(service.Status.LoadBalancer.Ingress) == 1 {
			// can/would this ever be more than 1? i dunno?
			address := service.Status.LoadBalancer.Ingress[0].IP
			if address != "" {
				objTopologyStatus.NodeExposedPorts[nodeName].LoadBalancerAddress = address
			}
		}

		err := ctrlruntimeutil.SetOwnerReference(obj, expectedService, r.Client.Scheme())
		if err != nil {
			return err
		}

		if !serviceConforms(service, expectedService, obj.GetUID()) {
			r.Log.Debugf(
				"comparing existing expose service '%s/%s' spec does not conform to desired "+
					"state, updating",
				service.Namespace,
				service.Name,
			)

			err = r.Client.Update(ctx, expectedService)
			if err != nil {
				r.Log.Criticalf(
					"failed updating expose service '%s/%s' error: %s",
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

func (r *Reconciler) parseContainerlabTopologyPortsSection(
	portDefinition string,
) (bool, *k8scorev1.ServicePort) {
	typedPort, err := clabernetesutilcontainerlab.ProcessPortDefinition(portDefinition)
	if err != nil {
		r.Log.Warnf("skipping port due to the following error: %s", err)

		return true, nil
	}

	return false, &k8scorev1.ServicePort{
		Name: fmt.Sprintf(
			"port-%d-%s", typedPort.DestinationPort, strings.ToLower(typedPort.Protocol),
		),
		Protocol: k8scorev1.Protocol(typedPort.Protocol),
		Port:     int32(typedPort.DestinationPort),
		TargetPort: intstr.IntOrString{
			IntVal: int32(typedPort.ExposePort),
		},
	}
}

func (r *Reconciler) renderExposeService(
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	objTopologyStatus clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) *k8scorev1.Service {
	configManager := r.ConfigManagerGetter()
	globalAnnotations, globalLabels := configManager.GetAllMetadata()

	name := obj.GetName()

	objTopologyStatus.NodeExposedPorts[nodeName] = &clabernetesapistopologyv1alpha1.ExposedPorts{
		TCPPorts: make([]int, 0),
		UDPPorts: make([]int, 0),
	}

	serviceName := fmt.Sprintf("%s-%s", name, nodeName)

	labels := map[string]string{
		clabernetesconstants.LabelApp:                 clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:                serviceName,
		clabernetesconstants.LabelTopologyOwner:       name,
		clabernetesconstants.LabelTopologyNode:        nodeName,
		clabernetesconstants.LabelTopologyKind:        r.ResourceKind,
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeExpose, //nolint:lll
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	ports := make([]k8scorev1.ServicePort, 0)

	// for actual containerlab configs we copy the users given defaults into each "sub topology" --
	// so in the case of containerlab we want to make sure we also iterate over the "default" or
	// topology wide ports that were specified. in this process we dont want to duplicate things, so
	// we use a simple set implementation to make sure we aren't doubling up on any port
	// definitions.
	allContainerlabPorts := clabernetesutil.NewStringSet()

	allContainerlabPorts.Extend(clabernetesConfigs[nodeName].Topology.Nodes[nodeName].Ports)

	allContainerlabPorts.Extend(clabernetesConfigs[nodeName].Topology.Defaults.Ports)

	allContainerlabPortsItems := allContainerlabPorts.Items()
	sort.Strings(allContainerlabPortsItems)

	for _, portDefinition := range allContainerlabPortsItems {
		shouldSkip, port := r.parseContainerlabTopologyPortsSection(portDefinition)

		if shouldSkip {
			continue
		}

		ports = append(ports, *port)

		// dont forget to update the exposed ports status bits
		if port.Protocol == clabernetesconstants.TCP {
			objTopologyStatus.NodeExposedPorts[nodeName].TCPPorts = append(
				objTopologyStatus.NodeExposedPorts[nodeName].TCPPorts,
				int(port.Port),
			)
		} else {
			objTopologyStatus.NodeExposedPorts[nodeName].UDPPorts = append(
				objTopologyStatus.NodeExposedPorts[nodeName].UDPPorts,
				int(port.Port),
			)
		}
	}

	service := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   obj.GetNamespace(),
			Annotations: globalAnnotations,
			Labels:      labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				clabernetesconstants.LabelTopologyOwner: name,
				clabernetesconstants.LabelTopologyNode:  nodeName,
			},
			Type: "LoadBalancer",
		},
	}

	return service
}
