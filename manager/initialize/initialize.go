package initialize

import (
	"fmt"

	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

// Initialize handles initialization tasks such as validating crds/webhook configurations.
func Initialize(c clabernetesmanagertypes.Clabernetes) {
	logger := c.GetBaseLogger()

	logger.Info("initializing certificates...")

	err := certificates(c)
	if err != nil {
		msg := fmt.Sprintf("failed initializing certificates, err: %s", err)

		logger.Critical(msg)

		clabernetesutil.Panic(msg)
	}

	logger.Debugf("initializing certificates complete...")

	logger.Info("initializing crds...")

	err = crds(c)
	if err != nil {
		msg := fmt.Sprintf("failed initializing crds, err: %s", err)

		logger.Critical(msg)

		clabernetesutil.Panic(msg)
	}

	logger.Debugf("initializing crds complete...")
}
