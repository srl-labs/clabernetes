package containerlab

import (
	"gopkg.in/yaml.v3"
)

// LoadContainerlabConfig loads a containerlab config definition from a raw containerlab config.
func LoadContainerlabConfig(rawConfig string) (*Config, error) {
	config := &Config{}

	err := yaml.Unmarshal([]byte(rawConfig), config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// LoadContainerlabTopology loads a containerlab topology definition from a raw containerlab config.
func LoadContainerlabTopology(rawConfig string) (*Topology, error) {
	// we don't care about the rest of the clab config, so just duplicate enough so we can
	// unmarshal and get the topology out
	config := &struct {
		Topology *Topology `json:"topology,omitempty"`
	}{}

	err := yaml.Unmarshal([]byte(rawConfig), config)
	if err != nil {
		return nil, err
	}

	if config.Topology.Defaults == nil {
		// defaults was nil, thats ok, but we'll just instantiate an empty definition so we don't
		// have to check that its nil before checking for stuff inside it being nil/empty too
		config.Topology.Defaults = &NodeDefinition{}
	}

	return config.Topology, nil
}
