package manager

import (
	"fmt"

	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

// prepare handles preparation tasks that happen before running the clabernetes.start method.
func (c *clabernetes) prepare() {
	c.logger.Info("begin prepare...")

	c.logger.Info("preparing certificates...")

	err := prepareCertificates(c)
	if err != nil {
		msg := fmt.Sprintf("failed preparing certificates, err: %s", err)

		c.logger.Critical(msg)

		clabernetesutil.Panic(msg)
	}

	c.logger.Debug("preparing certificates complete...")

	c.logger.Info("preparing scheme...")

	err = registerToScheme(c)
	if err != nil {
		msg := fmt.Sprintf("failed registering apis to scheme, err: %s", err)

		c.logger.Critical(msg)

		clabernetesutil.Panic(msg)
	}

	c.logger.Debug("preparing scheme complete...")
}
