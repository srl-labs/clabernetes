package util

import (
	containerlabtypes "github.com/srl-labs/containerlab/types"
	"gopkg.in/yaml.v3"
)

// LoadContainerlabTopology loads a containerlab topology definition from a raw containerlab config.
func LoadContainerlabTopology(rawConfig string) (*containerlabtypes.Topology, error) {
	// we don't care about the rest of the clab config, so just duplicate enough so we can
	// unmarshal and get the topology out
	config := &struct {
		Topology *containerlabtypes.Topology `json:"topology,omitempty"`
	}{}

	err := yaml.Unmarshal([]byte(rawConfig), config)
	if err != nil {
		return nil, err
	}

	return config.Topology, nil
}
