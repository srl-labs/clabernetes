package reconciler

import (
	"fmt"

	clabernetesutil "github.com/srl-labs/clabernetes/util"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NewServiceFabricReconciler returns an instance of ServiceFabricReconciler.
func NewServiceFabricReconciler(
	log claberneteslogging.Instance,
	owningTopologyKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *ServiceFabricReconciler {
	return &ServiceFabricReconciler{
		log:                 log,
		owningTopologyKind:  owningTopologyKind,
		configManagerGetter: configManagerGetter,
	}
}

// ServiceFabricReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating the "fabric" service for a
// clabernetes topology resource.
type ServiceFabricReconciler struct {
	log                 claberneteslogging.Instance
	owningTopologyKind  string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of services that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current services for the topology.
func (r *ServiceFabricReconciler) Resolve(
	ownedServices *k8scorev1.ServiceList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	_ clabernetesapistopologyv1alpha1.TopologyCommonObject,
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

	services.SetMissing(allNodes)
	services.SetExtra(allNodes)

	return services, nil
}

func (r *ServiceFabricReconciler) renderServiceBase(
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
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeFabric, //nolint:lll

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
			Ports: []k8scorev1.ServicePort{
				{
					Name:     "vxlan",
					Protocol: clabernetesconstants.UDP,
					Port:     clabernetesconstants.VXLANServicePort,
					TargetPort: intstr.IntOrString{
						IntVal: clabernetesconstants.VXLANServicePort,
					},
				},
			},
			Selector: selectorLabels,
			Type:     clabernetesconstants.KubernetesServiceClusterIPType,
		},
	}
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the final fabric service for this node.
func (r *ServiceFabricReconciler) Render(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	nodeName string,
) *k8scorev1.Service {
	owningTopologyName := owningTopology.GetName()

	service := r.renderServiceBase(
		fmt.Sprintf("%s-%s-vx", owningTopologyName, nodeName),
		owningTopology.GetNamespace(),
		owningTopologyName,
		nodeName,
	)

	return service
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the final fabric services for the given nodes.
func (r *ServiceFabricReconciler) RenderAll(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
	nodeNames []string,
) []*k8scorev1.Service {
	services := make([]*k8scorev1.Service, len(nodeNames))

	for idx, nodeName := range nodeNames {
		services[idx] = r.Render(
			owningTopology,
			nodeName,
		)
	}

	return services
}

// Conforms checks if the existingService conforms with the renderedService.
func (r *ServiceFabricReconciler) Conforms(
	existingService,
	renderedService *k8scorev1.Service,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	return ServiceConforms(existingService, renderedService, expectedOwnerUID)
}