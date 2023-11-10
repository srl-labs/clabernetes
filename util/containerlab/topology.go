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

	if config.Topology.Defaults == nil {
		// defaults was nil, thats ok, but we'll just instantiate an empty definition so we don't
		// have to check that its nil before checking for stuff inside it being nil/empty too
		config.Topology.Defaults = &NodeDefinition{}
	}

	return config, nil
}
