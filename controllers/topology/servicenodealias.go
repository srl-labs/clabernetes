package topology

import (
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// NewServiceNodeAliasReconciler returns an instance of ServiceNodeAliasReconciler.
func NewServiceNodeAliasReconciler(
	log claberneteslogging.Instance,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *ServiceNodeAliasReconciler {
	return &ServiceNodeAliasReconciler{
		log:                 log,
		configManagerGetter: configManagerGetter,
	}
}

// ServiceNodeAliasReconciler is a subcomponent of the "TopologyReconciler" but is exposed
// for testing purposes. This is the component responsible for rendering/validating the "node
// resolution" services for a clabernetes topology resource.
type ServiceNodeAliasReconciler struct {
	log                 claberneteslogging.Instance
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of services that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current services for the topology.
func (r *ServiceNodeAliasReconciler) Resolve(
	ownedServices *k8scorev1.ServiceList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	owningTopology *clabernetesapisv1alpha1.Topology,
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

		if topologyServiceType != clabernetesconstants.TopologyServiceTypeNodeAlias {
			// not the kind of service we're looking for here, we only care about the services
			// used for "node resolution" here.
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

	var allNodes []string

	if owningTopology.Spec.Expose.EnableNodeAliasService {
		allNodes = make([]string, len(clabernetesConfigs))

		idx := 0

		for nodeName := range clabernetesConfigs {
			allNodes[idx] = nodeName

			idx++
		}
	}

	services.SetMissing(allNodes)
	services.SetExtra(allNodes)

	return services, nil
}

func (r *ServiceNodeAliasReconciler) renderServiceBase(
	owningTopology *clabernetesapisv1alpha1.Topology,
	name,
	nodeName string,
) *k8scorev1.Service {
	owningTopologyName := owningTopology.GetName()
	owningTopologyNamespace := owningTopology.GetNamespace()

	annotations, globalLabels := r.configManagerGetter().GetAllMetadata()

	deploymentName := fmt.Sprintf("%s-%s", owningTopologyName, nodeName)

	selectorLabels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: owningTopologyName,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	labels := map[string]string{
		clabernetesconstants.LabelTopologyKind:        GetTopologyKind(owningTopology),
		clabernetesconstants.LabelTopologyServiceType: clabernetesconstants.TopologyServiceTypeNodeAlias, //nolint:lll

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
			Namespace:   owningTopologyNamespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: k8scorev1.ServiceSpec{
			Type: "ExternalName",
			ExternalName: fmt.Sprintf(
				"%s.%s.%s",
				name,
				owningTopologyNamespace,
				r.configManagerGetter().GetInClusterDNSSuffix(),
			),
			Selector: selectorLabels,
		},
	}
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the final expose service for this node.
func (r *ServiceNodeAliasReconciler) Render(
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeName string,
) *k8scorev1.Service {
	service := r.renderServiceBase(
		owningTopology,
		nodeName,
		nodeName,
	)

	return service
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the final node resolution services for the given nodes.
func (r *ServiceNodeAliasReconciler) RenderAll(
	owningTopology *clabernetesapisv1alpha1.Topology,
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
func (r *ServiceNodeAliasReconciler) Conforms(
	existingService,
	renderedService *k8scorev1.Service,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	return ServiceConforms(existingService, renderedService, expectedOwnerUID)
}
