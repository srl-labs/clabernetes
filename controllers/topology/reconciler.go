package topology

import (
	"context"
	"slices"

	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"

	"gopkg.in/yaml.v3"

	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler (TopologyReconciler) is the base clabernetes topology reconciler that is embedded in
// all clabernetes topology controllers, it provides common methods for reconciling the
// common/standard resources that represent a clabernetes object (configmap, deployments,
// services, etc.).
type Reconciler struct {
	Log          claberneteslogging.Instance
	Client       ctrlruntimeclient.Client
	ResourceKind string
}

// ReconcileConfigMap reconciles the primary configmap containing clabernetes configs and tunnel
// information.
func (r *Reconciler) ReconcileConfigMap(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	tunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) error {
	configMap := &k8scorev1.ConfigMap{}

	err := r.Client.Get(
		ctx,
		apimachinerytypes.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
		configMap,
	)
	if err != nil {
		if apimachineryerrors.IsNotFound(err) {
			return r.createConfigMap(ctx, obj, clabernetesConfigs, tunnels)
		}

		return err
	}

	return r.enforceConfigMap(ctx, obj, clabernetesConfigs, tunnels, configMap)
}

// ReconcileDeployments reconciles the deployments that make up a clabernetes Topology.
func (r *Reconciler) ReconcileDeployments(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	preReconcileConfigs,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) error {
	deployments, err := r.resolveDeployments(ctx, obj, clabernetesConfigs)
	if err != nil {
		return err
	}

	err = r.pruneDeployments(ctx, deployments)
	if err != nil {
		return err
	}

	err = r.enforceDeployments(ctx, obj, deployments)
	if err != nil {
		return err
	}

	nodesNeedingRestart := determineNodesNeedingRestart(preReconcileConfigs, clabernetesConfigs)
	if len(nodesNeedingRestart) == 0 {
		return nil
	}

	for _, nodeName := range nodesNeedingRestart {
		if slices.Contains(deployments.Missing, nodeName) {
			// is a new node, don't restart, we'll deploy it soon
			continue
		}

		r.Log.Infof(
			"restarting the nodes '%s' as configurations have changed",
			nodesNeedingRestart,
		)

		err = r.restartDeploymentForNode(ctx, obj, nodeName)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReconcileServiceFabric reconciles the service used for "fabric" (inter node) connectivity.
func (r *Reconciler) ReconcileServiceFabric(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) error {
	services, err := r.resolveFabricServices(ctx, obj, clabernetesConfigs)
	if err != nil {
		return err
	}

	err = r.pruneFabricServices(ctx, services)
	if err != nil {
		return err
	}

	err = r.enforceFabricServices(ctx, obj, services)
	if err != nil {
		return err
	}

	return nil
}

// ReconcileServicesExpose reconciles the service(s) used for exposing nodes.
func (r *Reconciler) ReconcileServicesExpose(
	ctx context.Context,
	obj clabernetesapistopologyv1alpha1.TopologyCommonObject,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
) (bool, error) {
	var shouldUpdate bool

	objTopologyStatus := obj.GetTopologyStatus()

	if objTopologyStatus.NodeExposedPorts == nil {
		objTopologyStatus.NodeExposedPorts = map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{} //nolint:lll

		shouldUpdate = true
	}

	services, err := r.resolveExposeServices(ctx, obj, clabernetesConfigs)
	if err != nil {
		return shouldUpdate, err
	}

	err = r.pruneExposeServices(ctx, services)
	if err != nil {
		return shouldUpdate, err
	}

	err = r.enforceExposeServices(ctx, obj, objTopologyStatus, clabernetesConfigs, services)
	if err != nil {
		return shouldUpdate, err
	}

	nodeExposedPortsBytes, err := yaml.Marshal(objTopologyStatus.NodeExposedPorts)
	if err != nil {
		return shouldUpdate, err
	}

	newNodeExposedPortsHash := clabernetesutil.HashBytes(nodeExposedPortsBytes)

	if objTopologyStatus.NodeExposedPortsHash != newNodeExposedPortsHash {
		objTopologyStatus.NodeExposedPortsHash = newNodeExposedPortsHash

		obj.SetTopologyStatus(objTopologyStatus)

		shouldUpdate = true
	}

	return shouldUpdate, nil
}
