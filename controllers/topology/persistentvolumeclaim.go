package topology

import (
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	clabernetesutilkubernetes "github.com/srl-labs/clabernetes/util/kubernetes"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clabernetesconfig "github.com/srl-labs/clabernetes/config"
	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	claberneteslogging "github.com/srl-labs/clabernetes/logging"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	k8scorev1 "k8s.io/api/core/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// NewPersistentVolumeClaimReconciler returns an instance of PersistentVolumeClaimReconciler.
func NewPersistentVolumeClaimReconciler(
	log claberneteslogging.Instance,
	configManagerGetter clabernetesconfig.ManagerGetterFunc,
) *PersistentVolumeClaimReconciler {
	return &PersistentVolumeClaimReconciler{
		log:                 log,
		configManagerGetter: configManagerGetter,
	}
}

// PersistentVolumeClaimReconciler is a subcomponent of the "TopologyReconciler" but is exposed for
// testing purposes. This is the component responsible for rendering/validating the optional PVC
// that is used to persist the containerlab directory of a topology's nodes.
type PersistentVolumeClaimReconciler struct {
	log                 claberneteslogging.Instance
	configManagerGetter clabernetesconfig.ManagerGetterFunc
}

// Resolve accepts a mapping of clabernetes configs and a list of services that are -- by owner
// reference and/or labels -- associated with the topology. It returns a ObjectDiffer object
// that contains the missing, extra, and current services for the topology.
func (r *PersistentVolumeClaimReconciler) Resolve(
	ownedPVCs *k8scorev1.PersistentVolumeClaimList,
	clabernetesConfigs map[string]*clabernetesutilcontainerlab.Config,
	owningTopology *clabernetesapisv1alpha1.Topology,
) (*clabernetesutil.ObjectDiffer[*k8scorev1.PersistentVolumeClaim], error) {
	pvcs := &clabernetesutil.ObjectDiffer[*k8scorev1.PersistentVolumeClaim]{
		Current: map[string]*k8scorev1.PersistentVolumeClaim{},
	}

	for i := range ownedPVCs.Items {
		labels := ownedPVCs.Items[i].Labels

		if labels == nil {
			return nil, fmt.Errorf(
				"%w: labels are nil, but we expect to see topology owner label here",
				claberneteserrors.ErrInvalidData,
			)
		}

		nodeName, ok := labels[clabernetesconstants.LabelTopologyNode]
		if !ok || nodeName == "" {
			return nil, fmt.Errorf(
				"%w: topology node label is missing or empty",
				claberneteserrors.ErrInvalidData,
			)
		}

		pvcs.Current[nodeName] = &ownedPVCs.Items[i]
	}

	persistenceEnabled := owningTopology.Spec.Persistence.Enabled

	if persistenceEnabled {
		allNodes := make([]string, len(clabernetesConfigs))

		var idx int

		for nodeName := range clabernetesConfigs {
			allNodes[idx] = nodeName

			idx++
		}

		pvcs.SetMissing(allNodes)
		pvcs.SetExtra(allNodes)
	} else {
		pvcs.SetExtra(nil)
	}

	return pvcs, nil
}

func (r *PersistentVolumeClaimReconciler) renderPVCBase(
	owningTopology *clabernetesapisv1alpha1.Topology,
	name,
	nodeName string,
) *k8scorev1.PersistentVolumeClaim {
	owningTopologyName := owningTopology.GetName()

	annotations, globalLabels := r.configManagerGetter().GetAllMetadata()

	deploymentName := fmt.Sprintf("%s-%s", owningTopologyName, nodeName)

	selectorLabels := map[string]string{
		clabernetesconstants.LabelApp:           clabernetesconstants.Clabernetes,
		clabernetesconstants.LabelName:          deploymentName,
		clabernetesconstants.LabelTopologyOwner: owningTopologyName,
		clabernetesconstants.LabelTopologyNode:  nodeName,
	}

	labels := map[string]string{
		clabernetesconstants.LabelTopologyKind: owningTopology.GetTopologyKind(),
	}

	for k, v := range selectorLabels {
		labels[k] = v
	}

	for k, v := range globalLabels {
		labels[k] = v
	}

	return &k8scorev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   owningTopology.GetNamespace(),
			Annotations: annotations,
			Labels:      labels,
		},
	}
}

func (r *PersistentVolumeClaimReconciler) renderPVCSpec(
	owningTopology *clabernetesapisv1alpha1.Topology,
	pvc *k8scorev1.PersistentVolumeClaim,
) {
	persistence := owningTopology.Spec.Persistence

	var storageClassName *string

	if persistence.StorageClassName != "" {
		storageClassName = clabernetesutil.ToPointer(persistence.StorageClassName)
	}

	pvcSize := resource.MustParse("5Gi")

	if persistence.ClaimSize != "" {
		userClaimSize, err := resource.ParseQuantity(persistence.ClaimSize)
		if err != nil {
			r.log.Warnf(
				"user provided claim size %q failed parsing, using default value instead",
				persistence.ClaimSize,
				err,
			)
		} else {
			pvcSize = userClaimSize
		}
	}

	pvc.Spec = k8scorev1.PersistentVolumeClaimSpec{
		AccessModes: []k8scorev1.PersistentVolumeAccessMode{
			k8scorev1.ReadWriteOnce,
		},
		Resources: k8scorev1.ResourceRequirements{
			Requests: k8scorev1.ResourceList{
				"storage": pvcSize,
			},
		},
		StorageClassName: storageClassName,
		VolumeMode:       clabernetesutil.ToPointer(k8scorev1.PersistentVolumeFilesystem),
	}
}

// Render accepts the owning topology a mapping of clabernetes sub-topology configs and a node name
// and renders the pvc for this node.
func (r *PersistentVolumeClaimReconciler) Render(
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeName string,
) *k8scorev1.PersistentVolumeClaim {
	owningTopologyName := owningTopology.GetName()

	pvc := r.renderPVCBase(
		owningTopology,
		fmt.Sprintf("%s-%s", owningTopologyName, nodeName),
		nodeName,
	)

	r.renderPVCSpec(owningTopology, pvc)

	return pvc
}

// RenderAll accepts the owning topology a mapping of clabernetes sub-topology configs and a
// list of node names and renders the pvcs for the given nodes.
func (r *PersistentVolumeClaimReconciler) RenderAll(
	owningTopology *clabernetesapisv1alpha1.Topology,
	nodeNames []string,
) []*k8scorev1.PersistentVolumeClaim {
	pvcs := make([]*k8scorev1.PersistentVolumeClaim, len(nodeNames))

	for idx, nodeName := range nodeNames {
		pvcs[idx] = r.Render(
			owningTopology,
			nodeName,
		)
	}

	return pvcs
}

// Conforms checks if the existingService conforms with the renderedService.
func (r *PersistentVolumeClaimReconciler) Conforms(
	existingPVC,
	renderedPVC *k8scorev1.PersistentVolumeClaim,
	expectedOwnerUID apimachinerytypes.UID,
) bool {
	existingClaimSize := existingPVC.Spec.Resources.Requests.Storage().Value()
	renderedClaimSize := renderedPVC.Spec.Resources.Requests.Storage().Value()

	if renderedClaimSize != existingClaimSize {
		if renderedClaimSize > existingClaimSize {
			// we only "dont conform" if the rendered claim size is greater than the existing claim;
			// we do this because we can *expand* but not shrink pvc claims
			return false
		}

		r.log.Warnf(
			"existing claim size of %q is *smaller* than desired claim size of %q,"+
				" however claim size can only be increased, not shrunk, ignoring...",
			existingPVC.Spec.Resources.Requests.Storage().String(),
			renderedPVC.Spec.Resources.Requests.Storage().String(),
		)
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingPVC.ObjectMeta.Annotations,
		renderedPVC.ObjectMeta.Annotations,
	) {
		return false
	}

	if !clabernetesutilkubernetes.AnnotationsOrLabelsConform(
		existingPVC.ObjectMeta.Labels,
		renderedPVC.ObjectMeta.Labels,
	) {
		return false
	}

	if len(existingPVC.ObjectMeta.OwnerReferences) != 1 {
		// we should have only one owner reference, the extractor
		return false
	}

	if existingPVC.ObjectMeta.OwnerReferences[0].UID != expectedOwnerUID {
		// owner ref uid is not us
		return false
	}

	// note: spec is immutable after creation so not bothering checking that

	return true
}
