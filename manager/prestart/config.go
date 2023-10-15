package prestart

import (
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
)

// config initializes the config manager singleton.
func config(c clabernetesmanagertypes.Clabernetes) error {
	clabernetesconfig.InitManager(
		c.GetContext(),
		c.GetAppName(),
		c.GetNamespace(),
		c.GetKubeClient(),
	)

	configManager := clabernetesconfig.GetManager()

	err := configManager.Start()
	if err != nil {
		return err
	}

	return nil
}
