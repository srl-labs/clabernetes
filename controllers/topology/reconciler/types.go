package reconciler

import (
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	"gopkg.in/yaml.v3"
)

// ReconcileData is a struct that holds data that is common during a reconciliation process
// regardless of the type of clabernetes topology that is being reconciled.
type ReconcileData struct {
	PreviousConfigsHash  string
	PreviousConfigs      map[string]*clabernetesutilcontainerlab.Config
	ResolvedConfigs      map[string]*clabernetesutilcontainerlab.Config
	ResolvedConfigsBytes []byte
	ResolvedConfigsHash  string

	PreviousTunnelsHash string
	PreviousTunnels     map[string][]*clabernetesapistopologyv1alpha1.Tunnel
	ResolvedTunnels     map[string][]*clabernetesapistopologyv1alpha1.Tunnel
	ResolvedTunnelsHash string

	ResolvedNodeExposedPorts     map[string]*clabernetesapistopologyv1alpha1.ExposedPorts
	ResolvedNodeExposedPortsHash string

	ShouldUpdateResource bool
	NodesNeedingReboot   clabernetesutil.StringSet
}

// NewReconcileData accepts a TopologyCommonObject and returns a ReconcileData object.
func NewReconcileData(
	owningTopology clabernetesapistopologyv1alpha1.TopologyCommonObject,
) (*ReconcileData, error) {
	status := owningTopology.GetTopologyStatus()

	rd := &ReconcileData{
		PreviousConfigsHash: status.ConfigsHash,
		PreviousConfigs:     make(map[string]*clabernetesutilcontainerlab.Config),
		ResolvedConfigs:     make(map[string]*clabernetesutilcontainerlab.Config),

		PreviousTunnelsHash: status.TunnelsHash,
		PreviousTunnels:     status.Tunnels,
		ResolvedTunnels:     make(map[string][]*clabernetesapistopologyv1alpha1.Tunnel),

		ResolvedNodeExposedPorts: map[string]*clabernetesapistopologyv1alpha1.ExposedPorts{},

		NodesNeedingReboot: clabernetesutil.NewStringSet(),
	}

	if status.Configs != "" {
		err := yaml.Unmarshal([]byte(status.Configs), &rd.PreviousConfigs)
		if err != nil {
			return nil, err
		}
	}

	return rd, nil
}

func (r *ReconcileData) SetStatus(
	owningTopologyStatus *clabernetesapistopologyv1alpha1.TopologyStatus,
) {
	owningTopologyStatus.Configs = string(r.ResolvedConfigsBytes)
	owningTopologyStatus.ConfigsHash = r.ResolvedConfigsHash
	owningTopologyStatus.Tunnels = r.ResolvedTunnels
	owningTopologyStatus.TunnelsHash = r.ResolvedTunnelsHash
	owningTopologyStatus.NodeExposedPorts = r.ResolvedNodeExposedPorts
	owningTopologyStatus.NodeExposedPortsHash = r.ResolvedNodeExposedPortsHash
}
