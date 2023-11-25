package reconciler

import (
	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

// NewPersistentVolumeClaimReconciler returns an instance of PersistentVolumeClaimReconciler.
func NewPersistentVolumeClaimReconciler(
	log claberneteslogging.Instance,
	owningTopologyKind string,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *PersistentVolumeClaimReconciler {
	return &PersistentVolumeClaimReconciler{
		log:                 log,
		owningTopologyKind:  owningTopologyKind,
		configManagerGetter: configManagerGetter,
	}
}

// PersistentVolumeClaimReconciler is a subcomponent of the "TopologyReconciler" but is exposed for
// testing purposes. This is the component responsible for rendering/validating the optional PVC
// that is used to persist the containerlab directory of a topology's nodes.
type PersistentVolumeClaimReconciler struct {
	log                 claberneteslogging.Instance
	owningTopologyKind  string
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}
