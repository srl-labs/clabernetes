package util

import (
	"fmt"

	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// MustSetupWithManager simply panics if registering a controller to the manager fails.
func MustSetupWithManager(setup func(mgr ctrlruntime.Manager) error, mgr ctrlruntime.Manager) {
	err := setup(mgr)
	if err != nil {
		Panic(
			fmt.Sprintf("failed registerring controller with manager, err: %s", err),
		)
	}
}
