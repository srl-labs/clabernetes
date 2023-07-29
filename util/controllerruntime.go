package util

import (
	"fmt"

	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrlruntime "sigs.k8s.io/controller-runtime"
)

// MustNewManager returns a new controller-runtime Manager or panics.
func MustNewManager(scheme *apimachineryruntime.Scheme) ctrlruntime.Manager {
	mgr, err := ctrlruntime.NewManager(
		ctrlruntime.GetConfigOrDie(),
		ctrlruntime.Options{
			Logger:             klog.NewKlogr(),
			Scheme:             scheme,
			MetricsBindAddress: "0",
			LeaderElection:     false,
		},
	)
	if err != nil {
		Panic(fmt.Sprintf("unable to start manager, error: %s", err))
	}

	return mgr
}

// MustSetupWithManager simply panics if registering a controller to the manager fails.
func MustSetupWithManager(setup func(mgr ctrlruntime.Manager) error, mgr ctrlruntime.Manager) {
	err := setup(mgr)
	if err != nil {
		Panic(
			fmt.Sprintf("failed registerring controller with manager, err: %s", err),
		)
	}
}
