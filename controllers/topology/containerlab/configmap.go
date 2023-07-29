package containerlab

import (
	"context"
	"fmt"
	"reflect"

	"github.com/srl-labs/containerlab/types"
	clabernetesapistopology "gitlab.com/carlmontanari/clabernetes/apis/topology"
	clabernetesconstants "gitlab.com/carlmontanari/clabernetes/constants"

	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	containerlabclab "github.com/srl-labs/containerlab/clab"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

func (c *Controller) reconcileConfigMap(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*containerlabclab.Config,
	tunnels map[string][]*clabernetesapistopology.Tunnel,
) error {
	configMap := &k8scorev1.ConfigMap{}

	err := c.BaseController.Client.Get(ctx, apimachinerytypes.NamespacedName{
		Name: clab.Name, Namespace: clab.Namespace,
	}, configMap)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return c.createConfigMap(ctx, clab, clabernetesConfigs, tunnels)
		}

		return err
	}

	return c.enforceConfigMap(ctx, clab, clabernetesConfigs, tunnels, configMap)
}

func (c *Controller) renderConfigMap(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*containerlabclab.Config,
	tunnels map[string][]*clabernetesapistopology.Tunnel,
) (*k8scorev1.ConfigMap, error) {
	configMap := &k8scorev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clab.Name,
			Namespace: clab.Namespace,
		},
		Data: map[string]string{},
	}

	for nodeName, nodeTopo := range clabernetesConfigs {
		if nodeTopo.Prefix == nil {
			p := clabernetesconstants.Clabernetes

			nodeTopo.Prefix = &p
		}

		if nodeTopo.Mgmt == nil {
			nodeTopo.Mgmt = &types.MgmtNet{}
		}

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

func (c *Controller) createConfigMap(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*containerlabclab.Config,
	tunnels map[string][]*clabernetesapistopology.Tunnel,
) error {
	configMap, err := c.renderConfigMap(clab, clabernetesConfigs, tunnels)
	if err != nil {
		return err
	}

	err = c.enforceConfigMapOwnerReference(clab, configMap)
	if err != nil {
		return err
	}

	return c.BaseController.Client.Create(ctx, configMap)
}

func (c *Controller) enforceConfigMap(
	ctx context.Context,
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	clabernetesConfigs map[string]*containerlabclab.Config,
	tunnels map[string][]*clabernetesapistopology.Tunnel,
	actual *k8scorev1.ConfigMap,
) error {
	configMap, err := c.renderConfigMap(clab, clabernetesConfigs, tunnels)
	if err != nil {
		return err
	}

	err = c.enforceConfigMapOwnerReference(clab, configMap)
	if err != nil {
		return err
	}

	if reflect.DeepEqual(actual.BinaryData, configMap.BinaryData) {
		// nothing to do
		return nil
	}

	return c.BaseController.Client.Update(ctx, configMap)
}

func (c *Controller) enforceConfigMapOwnerReference(
	clab *clabernetesapistopologyv1alpha1.Containerlab,
	configMap *k8scorev1.ConfigMap,
) error {
	err := ctrlruntimeutil.SetOwnerReference(clab, configMap, c.BaseController.Client.Scheme())
	if err != nil {
		c.BaseController.Log.Criticalf(
			"failed setting owner reference on configMap '%s/%s' error: %s",
			configMap.Namespace,
			configMap.Name,
			err,
		)

		return err
	}

	return nil
}
