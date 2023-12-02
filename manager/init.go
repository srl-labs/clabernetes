package manager

import (
	"context"
	"fmt"
)

func (c *clabernetes) init(ctx context.Context) {
	c.logger.Info("begin init...")

	c.leaderCtx = ctx

	c.logger.Info("initializing certificates...")

	err := initializeCertificates(c)
	if err != nil {
		msg := fmt.Sprintf("failed initializing certificates, err: %s", err)

		c.logger.Critical(msg)

		c.Panic(err.Error())
	}

	c.logger.Debugf("initializing certificates complete...")

	c.logger.Info("initializing crds...")

	err = initCrds(c)
	if err != nil {
		msg := fmt.Sprintf("failed initializing crds, err: %s", err)

		c.logger.Critical(msg)

		c.Panic(err.Error())
	}

	c.logger.Debugf("initializing crds complete...")

	c.logger.Info("init complete...")

	c.baseCtxCancel()
}
