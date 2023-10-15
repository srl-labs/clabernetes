package prestart

import (
	"fmt"

	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
)

// PreStart handles tasks after initialization/preparation, but *before* starting controller-runtime
// manager.
func PreStart(c clabernetesmanagertypes.Clabernetes) {
	logger := c.GetBaseLogger()

	logger.Info("starting config manager...")

	err := config(c)
	if err != nil {
		// we *shouldn't* actually ever hit this as the config manager can start and *not* find a
		// config that it manages just fine, but i guess its possible that something terrible
		// could happen that would prevent us from continuing.
		msg := fmt.Sprintf("failed starting config manager, err: %s", err)

		logger.Critical(msg)

		clabernetesutil.Panic(msg)
	}

	logger.Debug("config manager started...")
}
