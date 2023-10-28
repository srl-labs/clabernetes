package reconciler

import (
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
)

// ReconcileData is a struct that holds data that is common during a reconciliation process
// regardless of the type of clabernetes topology that is being reconciled.
type ReconcileData struct {
	PreReconcileConfigsHash   string
	PreReconcileConfigs       map[string]*clabernetesutilcontainerlab.Config
	PostReconcileConfigs      map[string]*clabernetesutilcontainerlab.Config
	PostReconcileConfigsBytes []byte
	PostReconcileConfigsHash  string

	PreReconcileTunnelsHash  string
	PreReconcileTunnels      map[string][]*clabernetesapistopologyv1alpha1.Tunnel
	PostReconcileTunnels     map[string][]*clabernetesapistopologyv1alpha1.Tunnel
	PostReconcileTunnelsHash string

	ShouldUpdateResource bool
}

// NewReconcileData accepts a TopologyCommonObject and returns a ReconcileData object.
func NewReconcileData(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
) (*ReconcileData, error) {
	status := owningTopology.GetTopologyStatus()

	rd := &ReconcileData{
		PreReconcileConfigsHash: status.ConfigsHash,
		PreReconcileConfigs:     make(map[string]*clabernetesutilcontainerlab.Config),
		PostReconcileConfigs:    make(map[string]*clabernetesutilcontainerlab.Config),

		PreReconcileTunnelsHash: status.TunnelsHash,
		PreReconcileTunnels:     status.Tunnels,
		PostReconcileTunnels:    make(map[string][]*clabernetesapistopologyv1alpha1.Tunnel),
	}

	if status.Configs != "" {
		err := yaml.Unmarshal([]byte(status.Configs), &rd.PreReconcileConfigs)
		if err != nil {
			return nil, err
		}
	}

	return rd, nil
}
