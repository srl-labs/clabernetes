package image

import (
	"fmt"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
)

// NewManager returns an image Manager for the given cri.
func NewManager(logger claberneteslogging.Instance, criKind string) (Manager, error) {
	switch criKind {
	case clabernetesconstants.KubernetesCRIContainerd:
		return &containerdManager{
			logger: logger,
		}, nil
	default:
		return nil, fmt.Errorf(
			"%w: unknown criKind, cannot create image manager",
			claberneteserrors.ErrLaunch,
		)
	}
}

// Manager is an interface defining an image manager -- basically a small abstraction around cri
// such that we can check for images, pull images, and export images from the cri sock mounted in
// a launcher pod.
type Manager interface {
	// Present checks if the image is already present in the node.
	Present(imageName string) (bool, error)
	// Export is the main reason we are using this and not the cri interface directly (cri has no
	// service for export!) -- and does what it says: exports an image to disk.
	Export(imageName, destination string) error
}
