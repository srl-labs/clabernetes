package topology

import (
	"context"
	"fmt"
	"reflect"

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
func RenderConfigMap(
	obj ctrlruntimeclient.Object,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) (*k8scorev1.ConfigMap, error) {
	configMap := &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
		Data: map[string]string{},
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
	configMap, err := RenderConfigMap(obj, clabernetesConfigs, tunnels)
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
	configMap, err := RenderConfigMap(obj, clabernetesConfigs, tunnels)
	if err != nil {
		return err
	}

	err = ctrlruntimeutil.SetOwnerReference(obj, configMap, r.Client.Scheme())
	if err != nil {
		return err
	}

	if reflect.DeepEqual(actual.BinaryData, configMap.BinaryData) {
		// nothing to do
		return nil
	}

	return r.Client.Update(ctx, configMap)
}
