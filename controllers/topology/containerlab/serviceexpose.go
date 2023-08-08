package containerlab

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	clabernetescontainerlab "gitlab.com/carlmontanari/clabernetes/containerlab"
	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	portPattern     *regexp.Regexp //nolint:gochecknoglobals
	portPatternOnce sync.Once      //nolint:gochecknoglobals
)

func getPortPattern() *regexp.Regexp {
	portPatternOnce.Do(func() {
		portPattern = regexp.MustCompile(
			`(?P<exposePort>\d+):(?P<destinationPort>\d+)/?(?P<protocol>(TCP)|(UDP))?`,
		)
	})

	return portPattern
}

func (c *Controller) reconcileExposeServices(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*clabernetescontainerlab.Config,
) (bool, error) {
	var shouldUpdate bool

	if clab.Status.NodeExposedPorts == nil {
		clab.Status.NodeExposedPorts = map[string]*clabernetesapistopology.ExposedPorts{}

		shouldUpdate = true
	}

	services, err := c.resolveExposeServices(ctx, clab, clabernetesConfigs)
	if err != nil {
		return shouldUpdate, err
	}

	err = c.pruneExposeServices(ctx, services)
	if err != nil {
		return shouldUpdate, err
	}

	err = c.enforceExposeServices(ctx, clab, clabernetesConfigs, services)
	if err != nil {
		return shouldUpdate, err
	}

	nodeExposedPortsBytes, err := yaml.Marshal(clab.Status.NodeExposedPorts)
	if err != nil {
		return shouldUpdate, err
	}

	newNodeExposedPortsHash := clabernetesutil.HashBytes(nodeExposedPortsBytes)

	if clab.Status.NodeExposedPortsHash != newNodeExposedPortsHash {
		clab.Status.NodeExposedPortsHash = newNodeExposedPortsHash

		shouldUpdate = true
	}

	return shouldUpdate, nil
}

func (c *Controller) resolveExposeServices(
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
		c.Log.Criticalf("failed fetching owned expose services, error: '%s'", err)

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

	exposedNodes := make([]string, 0)

	for nodeName, nodeData := range clabernetesConfigs {
		if len(nodeData.Topology.Nodes[nodeName].Ports) == 0 {
			continue
		}

		exposedNodes = append(exposedNodes, nodeName)
	}

	services.Missing = clabernetesutil.StringSliceDifference(
		services.CurrentServiceNames(),
		exposedNodes,
	)

	c.BaseController.Log.Debugf(
		"expose services are missing for the following nodes: %s",
		services.Missing,
	)

	extraEndpointDeployments := clabernetesutil.StringSliceDifference(
		exposedNodes,
		services.CurrentServiceNames(),
	)

	c.BaseController.Log.Debugf(
		"extraneous expose services exist for following nodes: %s",
		extraEndpointDeployments,
	)

	services.Extra = make([]*k8scorev1.Service, len(extraEndpointDeployments))

	for idx, endpoint := range extraEndpointDeployments {
		services.Extra[idx] = services.Current[endpoint]
	}

	return services, nil
}

func (c *Controller) pruneExposeServices(
	ctx context.Context,
	services *clabernetescontrollers.ResolvedServices,
) error {
	c.BaseController.Log.Info("pruning extraneous expose services")

	for _, extraDeployment := range services.Extra {
		c.BaseController.Log.Debugf(
			"removing expose service '%s/%s'",
			extraDeployment.Namespace,
			extraDeployment.Name,
		)

		err := c.Client.Delete(ctx, extraDeployment)
		if err != nil {
			c.BaseController.Log.Criticalf(
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

func (c *Controller) enforceExposeServices(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*clabernetescontainerlab.Config,
	services *clabernetescontrollers.ResolvedServices,
) error {
	c.BaseController.Log.Info("creating missing expose services")

	for _, nodeName := range services.Missing {
		service := c.renderExposeService(
			clab,
			clabernetesConfigs,
			nodeName,
		)

		err := c.enforceServiceOwnerReference(clab, service)
		if err != nil {
			return err
		}

		c.BaseController.Log.Debugf(
			"creating expose service '%s/%s'",
			service.Namespace,
			service.Name,
		)

		err = c.Client.Create(ctx, service)
		if err != nil {
			c.BaseController.Log.Criticalf(
				"failed creating expose service '%s/%s' error: %s",
				service.Namespace,
				service.Name,
				err,
			)

			return err
		}
	}

	// compare and update existing deployments if we need to
	c.BaseController.Log.Info("enforcing desired state on existing expose services")

	for nodeName, service := range services.Current {
		c.BaseController.Log.Debugf(
			"comparing existing expose service '%s/%s' to desired state",
			service.Namespace,
			service.Name,
		)

		expectedService := c.renderExposeService(
			clab,
			clabernetesConfigs,
			nodeName,
		)

		if len(service.Status.LoadBalancer.Ingress) == 1 {
			// can/would this ever be more than 1? i dunno?
			address := service.Status.LoadBalancer.Ingress[0].IP
			if address != "" {
				clab.Status.NodeExposedPorts[nodeName].LoadBalancerAddress = address
			}
		}

		err := c.enforceServiceOwnerReference(clab, expectedService)
		if err != nil {
			return err
		}

		if !serviceConforms(service, expectedService, clab.UID) {
			c.BaseController.Log.Debugf(
				"comparing existing expose service '%s/%s' spec does not conform to desired "+
					"state, updating",
				service.Namespace,
				service.Name,
			)

			err = c.Client.Update(ctx, expectedService)
			if err != nil {
				c.BaseController.Log.Criticalf(
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

func (c *Controller) renderExposeService(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*clabernetescontainerlab.Config,
	nodeName string,
) *k8scorev1.Service {
	clab.Status.NodeExposedPorts[nodeName] = &clabernetesapistopology.ExposedPorts{
		TCPPorts: make([]int, 0),
		UDPPorts: make([]int, 0),
	}

	serviceName := fmt.Sprintf("%s-%s-expose", clab.Name, nodeName)

	labels := map[string]string{
		clabernetesconstants.LabelApp:                 clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:                serviceName,
		clabernetesconstants.LabelTopologyOwner:       clab.Name,
		clabernetesconstants.LabelTopologyNode:        nodeName,
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeExpose, //nolint:lll
	}

	ports := make([]k8scorev1.ServicePort, 0)

	re := getPortPattern()

	for _, portDefinition := range clabernetesConfigs[nodeName].Topology.Nodes[nodeName].Ports {
		portDefinition = strings.ToUpper(portDefinition)

		paramsMap := clabernetesutil.RegexStringSubMatchToMap(re, portDefinition)

		protocol := clabernetesconstants.TCP
		if paramsMap["protocol"] == clabernetesconstants.UDP {
			protocol = clabernetesconstants.UDP
		}

		exposePortAsInt, err := strconv.ParseInt(paramsMap["exposePort"], 10, 32)
		if err != nil || exposePortAsInt == 0 {
			c.BaseController.Log.Warnf(
				"failed converting exposed port to integer, full port string '%s', parsed port "+
					"'%s'. skipping this port but continuing on...",
				portDefinition,
				paramsMap["exposePort"],
			)

			continue
		}

		destinationPortAsInt, err := strconv.ParseInt(paramsMap["destinationPort"], 10, 32)
		if err != nil || destinationPortAsInt == 0 {
			c.BaseController.Log.Warnf(
				"failed converting destination port to integer, full port string '%s', parsed "+
					"port '%s'.  skipping this port but continuing on...",
				portDefinition,
				paramsMap["destinationPort"],
			)

			continue
		}

		ports = append(
			ports,
			k8scorev1.ServicePort{
				Name: fmt.Sprintf(
					"port-%s-%s", paramsMap["destinationPort"], strings.ToLower(protocol),
				),
				Protocol: k8scorev1.Protocol(protocol),
				Port:     int32(destinationPortAsInt),
				TargetPort: intstr.IntOrString{
					IntVal: int32(exposePortAsInt),
				},
			},
		)

		// dont forget to update the exposed ports status bits
		if protocol == clabernetesconstants.TCP {
			clab.Status.NodeExposedPorts[nodeName].TCPPorts = append(
				clab.Status.NodeExposedPorts[nodeName].TCPPorts,
				int(destinationPortAsInt),
			)
		} else {
			clab.Status.NodeExposedPorts[nodeName].UDPPorts = append(
				clab.Status.NodeExposedPorts[nodeName].UDPPorts,
				int(destinationPortAsInt),
			)
		}
	}

	service := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: clab.Namespace,
			Labels:    labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Ports: ports,
			Selector: map[string]string{
				clabernetesconstants.LabelTopologyOwner: clab.Name,
				clabernetesconstants.LabelTopologyNode:  nodeName,
			},
			Type: "LoadBalancer",
		},
	}

	return service
}
