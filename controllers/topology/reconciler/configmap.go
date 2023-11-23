package reconciler

import (
	"fmt"
	"reflect"

	claberneteslogging "github.com/srl-labs/clabernetes/logging"

	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// NewConfigMapReconciler returns an instance of ConfigMapReconciler.
func NewConfigMapReconciler(
	log claberneteslogging.Instance,
	owningTopologyKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *ConfigMapReconciler {
	return &ConfigMapReconciler{
		log:                 log,
		owningTopologyKind:  owningTopologyKind,
		configManagerGetter: configManagerGetter,
	}
}

// ConfigMapReconciler is a subcomponent of the "TopologyReconciler" but is exposed for testing
// purposes. This is the component responsible for rendering/validating configmaps for a
// clabernetes topology resource.
type ConfigMapReconciler struct {
	log                 claberneteslogging.Instance
	owningTopologyKind  string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Render accepts an object (just for name/namespace reasons) and a mapping of clabernetes
// sub-topology configs and tunnels and renders the final configmap for the deployment -- this is
// the configmap that will ultimately be referenced when mounting sub-topologies and tunnel data in
// the clabernetes launcher pod(s) for a given topology.
func (r *ConfigMapReconciler) Render(
	owningTopologyNamespacedName apimachinerytypes.NamespacedName,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
	filesFromURL map[string][]clabernetesapistopologyv1alpha1.FileFromURL,
) (*k8scorev1.ConfigMap, error) {
	annotations, globalLabels := r.configManagerGetter().GetAllMetadata()

	labels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          owningTopologyNamespacedName.Name,
		clabernetesconstants.LabelTopologyOwner: owningTopologyNamespacedName.Name,
		clabernetesconstants.LabelTopologyKind:  r.owningTopologyKind,
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	data := map[string]string{
		// we always make this key like the other keys so we can be lazy and not have to wonder if
		// the key / mounted file exists.
		"configured-pull-secrets": "",
	}

	for nodeName, nodeTopo := range clabernetesConfigs {
		// always initialize the tunnels and files from url keys in the configmap, this way we don't
		// have to have any special handling for no tunnels and things always look consistent;
		// we'll override this down below if the node has tunnels of course!
		data[fmt.Sprintf("%s-tunnels", nodeName)] = ""
		data[fmt.Sprintf("%s-files-from-url", nodeName)] = ""

		yamlNodeTopo, err := yaml.Marshal(nodeTopo)
		if err != nil {
			return nil, err
		}

		data[nodeName] = string(yamlNodeTopo)
	}

	for nodeName, nodeTunnels := range tunnels {
		yamlNodeTunnels, err := yaml.Marshal(nodeTunnels)
		if err != nil {
			return nil, err
		}

		data[fmt.Sprintf("%s-tunnels", nodeName)] = string(yamlNodeTunnels)
	}

	for nodeName, nodeFilesFromURL := range filesFromURL {
		// ignore bad node names
		_, nodeOk := clabernetesConfigs[nodeName]
		if !nodeOk {
			continue
		}

		yamlNodeFilesFromURL, err := yaml.Marshal(nodeFilesFromURL)
		if err != nil {
			return nil, err
		}

		data[fmt.Sprintf("%s-files-from-url", nodeName)] = string(yamlNodeFilesFromURL)
	}

	return &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        owningTopologyNamespacedName.Name,
			Namespace:   owningTopologyNamespacedName.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Data: data,
	}, nil
}

// Conforms checks if the existingConfigMap conforms with the renderedConfigMap.
func (r *ConfigMapReconciler) Conforms(
	existingConfigMap,
	renderedConfigMap *k8scorev1.ConfigMap,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	if !reflect.DeepEqual(existingConfigMap.Data, renderedConfigMap.Data) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingConfigMap.ObjectMeta.Annotations,
		renderedConfigMap.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingConfigMap.ObjectMeta.Labels,
		renderedConfigMap.ObjectMeta.Labels,
	) {
		return false
	}

	if len(existingConfigMap.ObjectMeta.OwnerReferences) != 1 {
		// we should have only one owner reference, the extractor
		return false
	}

	if existingConfigMap.ObjectMeta.OwnerReferences[0].UID != expectedOwnerUID {
		// owner ref uid is not us
		return false
	}

	return true
}
