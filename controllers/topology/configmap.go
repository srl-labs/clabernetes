package topology

import (
	"context"
	"fmt"
	"reflect"

	clabernetescontrollers "github.com/srl-labs/clabernetes/controllers"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RenderConfigMap accepts an object (just for name/namespace reasons) and a mapping of clabernetes
// sub-topology configs and tunnels and renders the final configmap for the deployment -- this is
// the configmap that will ultimately be referenced when mounting sub-topologies and tunnel data in
// the clabernetes launcher pod(s) for a given topology.
func (r *Reconciler) RenderConfigMap(
	obj ctrlruntimeclient.Object,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) (*k8scorev1.ConfigMap, error) {
	configManager := r.ConfigManagerGetter()
	globalAnnotations, globalLabels := configManager.GetAllMetadata()

	configMapName := obj.GetName()

	configMap := &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Namespace:   obj.GetNamespace(),
			Annotations: globalAnnotations,
			Labels: map[string]string{
				clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
				clabernetesconstants.LabelName:          configMapName,
				clabernetesconstants.LabelTopologyOwner: configMapName,
				clabernetesconstants.LabelTopologyKind:  r.ResourceKind,
			},
		},
		Data: map[string]string{},
	}

	for k, v := range globalLabels {
		configMap.Labels[k] = v
	}

	for nodeName, nodeTopo := range clabernetesConfigs {
		// always initialize the tunnels keys in the configmap, this way we don't have to have any
		// special handling for no tunnels and things always look consistent; we'll override this
		// down below if the node has tunnels of course!
		configMap.Data[fmt.Sprintf("%s-tunnels", nodeName)] = ""

		yamlNodeTopo, err := yaml.Marshal(nodeTopo)
		if err != nil {
			return nil, err
		}

		configMap.Data[nodeName] = string(yamlNodeTopo)
	}

	for nodeName, nodeTunnels := range tunnels {
		yamlNodeTunnels, err := yaml.Marshal(nodeTunnels)
		if err != nil {
			return nil, err
		}

		configMap.Data[fmt.Sprintf("%s-tunnels", nodeName)] = string(yamlNodeTunnels)
	}

	return configMap, nil
}

func (r *Reconciler) createConfigMap(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) error {
	configMap, err := r.RenderConfigMap(obj, clabernetesConfigs, tunnels)
	if err != nil {
		return err
	}

	err = ctrlruntimeutil.SetOwnerReference(obj, configMap, r.Client.Scheme())
	if err != nil {
		return err
	}

	return r.Client.Create(ctx, configMap)
}

func (r *Reconciler) enforceConfigMap(
	ctx context.Context,
	obj ctrlruntimeclient.Object,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
	actual *k8scorev1.ConfigMap,
) error {
	configMap, err := r.RenderConfigMap(obj, clabernetesConfigs, tunnels)
	if err != nil {
		return err
	}

	err = ctrlruntimeutil.SetOwnerReference(obj, configMap, r.Client.Scheme())
	if err != nil {
		return err
	}

	if configMapConforms(actual, configMap, obj.GetUID()) {
		// nothing to do
		return nil
	}

	return r.Client.Update(ctx, configMap)
}

func configMapConforms(
	existingConfigMap,
	renderedConfigMap *k8scorev1.ConfigMap,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	if !reflect.DeepEqual(existingConfigMap.Data, renderedConfigMap.Data) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
		existingConfigMap.ObjectMeta.Annotations,
		renderedConfigMap.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetescontrollers.AnnotationsOrLabelsConform(
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
