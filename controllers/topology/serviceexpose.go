package topology

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	clabernetesutilcontainerlab "gitlab.com/carlmontanari/clabernetes/util/containerlab"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"
	clabernetescontrollers "gitlab.com/carlmontanari/clabernetes/controllers"
	claberneteserrors "gitlab.com/carlmontanari/clabernetes/errors"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (r *Reconciler) renderExposeService(
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	objTopologyStatus clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) *k8scorev1.Service {
	name := obj.GetName()

	objTopologyStatus.NodeExposedPorts[nodeName] = &clabernetesapistopologyv1alpha1.ExposedPorts{
		TCPPorts: make([]int, 0),
		UDPPorts: make([]int, 0),
	}

	serviceName := fmt.Sprintf("%s-%s-expose", name, nodeName)

	labels := map[string]string{
		clabernetesconstants.LabelApp:                 clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:                serviceName,
		clabernetesconstants.LabelTopologyOwner:       name,
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
			r.Log.Warnf(
				"failed converting exposed port to integer, full port string '%s', parsed port "+
					"'%s'. skipping this port but continuing on...",
				portDefinition,
				paramsMap["exposePort"],
			)

			continue
		}

		destinationPortAsInt, err := strconv.ParseInt(paramsMap["destinationPort"], 10, 32)
		if err != nil || destinationPortAsInt == 0 {
			r.Log.Warnf(
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
			objTopologyStatus.NodeExposedPorts[nodeName].TCPPorts = append(
				objTopologyStatus.NodeExposedPorts[nodeName].TCPPorts,
				int(destinationPortAsInt),
			)
		} else {
			objTopologyStatus.NodeExposedPorts[nodeName].UDPPorts = append(
				objTopologyStatus.NodeExposedPorts[nodeName].UDPPorts,
				int(destinationPortAsInt),
			)
		}
	}

	service := &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: obj.GetNamespace(),
			Labels:    labels,
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
