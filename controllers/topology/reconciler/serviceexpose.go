package reconciler

import (
	"fmt"
	"sort"
	"strings"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8scorev1 "k8s.io/api/core/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// NewServiceExposeReconciler returns an instance of ServiceExposeReconciler.
func NewServiceExposeReconciler(
	log claberneteslogging.Instance,
	owningTopologyKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *ServiceExposeReconciler {
	return &ServiceExposeReconciler{
		log:                 log,
		owningTopologyKind:  owningTopologyKind,
		configManagerGetter: configManagerGetter,
	}
}

// ServiceExposeReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating the "expose" service for a
// clabernetes topology resource.
type ServiceExposeReconciler struct {
	log                 claberneteslogging.Instance
	owningTopologyKind  string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of services that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current services for the topology.
func (r *ServiceExposeReconciler) Resolve(
	ownedServices *k8scorev1.ServiceList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
) (*clabernetesutil.ObjectDiffer[*k8scorev1.Service], error) {
	services := &clabernetesutil.ObjectDiffer[*k8scorev1.Service]{
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

	commonSpec := owningTopology.GetTopologyCommonSpec()
	disableExpose := commonSpec.DisableExpose
	disableAutoExpose := commonSpec.DisableAutoExpose

	for nodeName, nodeData := range clabernetesConfigs {
		// disable expose is set to true for the whole spec, nothing should be exposed, so skip
		// every node
		if disableExpose {
			continue
		}

		// if disable auto expose is true *and* there are no ports defined for the node *and*
		// there are no default ports defined for the topology we can skip the node from an expose
		// perspective.
		if disableAutoExpose &&
			len(nodeData.Topology.Nodes[nodeName].Ports) == 0 &&
			len(nodeData.Topology.Defaults.Ports) == 0 {
			continue
		}

		exposedNodes = append(exposedNodes, nodeName)
	}

	services.SetMissing(exposedNodes)
	services.SetExtra(exposedNodes)

	return services, nil
}

func (r *ServiceExposeReconciler) renderServiceBase(
	name,
	namespace,
	owningTopologyName,
	nodeName string,
) *k8scorev1.Service {
	annotations, globalLabels := r.configManagerGetter().GetAllMetadata()

	deploymentName := fmt.Sprintf("%s-%s", owningTopologyName, nodeName)

	selectorLabels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: owningTopologyName,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	labels := map[string]string{
		clabernetesconstants.LabelTopologyKind:        r.owningTopologyKind,
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeExpose, //nolint:lll

	}

	for k, v := range selectorLabels {
		labels[k] = v
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	return &k8scorev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Selector: selectorLabels,
			Type:     clabernetesconstants.KubernetesServiceLoadBalancerType,
		},
	}
}

func (r *ServiceExposeReconciler) parseContainerlabTopologyPortsSection(
	portDefinition string,
) (bool, *k8scorev1.ServicePort) {
	typedPort, err := clabernetesutilcontainerlab.ProcessPortDefinition(portDefinition)
	if err != nil {
		r.log.Warnf("skipping port due to the following error: %s", err)

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

func (r *ServiceExposeReconciler) renderServicePorts(
	service *k8scorev1.Service,
	owningTopologyStatus *clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) {
	owningTopologyStatus.NodeExposedPorts[nodeName] = &clabernetesapistopologyv1alpha1.ExposedPorts{
		TCPPorts: make([]int, 0),
		UDPPorts: make([]int, 0),
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
			owningTopologyStatus.NodeExposedPorts[nodeName].TCPPorts = append(
				owningTopologyStatus.NodeExposedPorts[nodeName].TCPPorts,
				int(port.Port),
			)
		} else {
			owningTopologyStatus.NodeExposedPorts[nodeName].UDPPorts = append(
				owningTopologyStatus.NodeExposedPorts[nodeName].UDPPorts,
				int(port.Port),
			)
		}
	}

	service.Spec.Ports = ports
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the final expose service for this node.
func (r *ServiceExposeReconciler) Render(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	owningTopologyStatus *clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeName string,
) *k8scorev1.Service {
	owningTopologyName := owningTopology.GetName()

	service := r.renderServiceBase(
		fmt.Sprintf("%s-%s", owningTopologyName, nodeName),
		owningTopology.GetNamespace(),
		owningTopologyName,
		nodeName,
	)

	r.renderServicePorts(
		service,
		owningTopologyStatus,
		clabernetesConfigs,
		nodeName,
	)

	return service
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the final expose services for the given nodes.
func (r *ServiceExposeReconciler) RenderAll(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	owningTopologyStatus *clabernetesapistopologyv1alpha1.TopologyStatus,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	nodeNames []string,
) []*k8scorev1.Service {
	services := make([]*k8scorev1.Service, len(nodeNames))

	for idx, nodeName := range nodeNames {
		services[idx] = r.Render(
			owningTopology,
			owningTopologyStatus,
			clabernetesConfigs,
			nodeName,
		)
	}

	return services
}

// Conforms checks if the existingService conforms with the renderedService.
func (r *ServiceExposeReconciler) Conforms(
	existingService,
	renderedService *k8scorev1.Service,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	return ServiceConforms(existingService, renderedService, expectedOwnerUID)
}
