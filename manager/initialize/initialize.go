package initialize

import (
	"fmt"

	clabernetesmanagertypes "gitlab.com/carlmontanari/clabernetes/manager/types"
	clabernetesutil "gitlab.com/carlmontanari/clabernetes/util"
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
}
