package topology

import (
	"bytes"
	"context"
	"fmt"

	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutil "github.com/srl-labs/clabernetes/util"
	k8scorev1 "k8s.io/api/core/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// defaultDefinitionRefConfigMapKey is the ConfigMap key read when a DefinitionRef omits ConfigMapKey.
const defaultDefinitionRefConfigMapKey = "containerlab"

// resolveDefinition returns the Topology to use for definition processing and expansion. When the
// Topology sources its containerlab definition indirectly (spec.definition.containerlabRef), this
// returns a *deep copy* with the resolved definition inlined -- so the rest of the reconcile pipeline
// is entirely unchanged -- while leaving the original (small-spec) Topology untouched. The original
// is what gets persisted, so the (potentially huge) raw definition is never written back onto the
// Topology object. When no ref is set the original Topology is returned unchanged.
func (c *Controller) resolveDefinition(
	ctx context.Context,
	topology *clabernetesapisv1alpha1.Topology,
) (*clabernetesapisv1alpha1.Topology, error) {
	ref := topology.Spec.Definition.ContainerlabRef
	if ref == nil {
		return topology, nil
	}

	if topology.Spec.Definition.Containerlab != "" {
		return nil, fmt.Errorf(
			"%w: both spec.definition.containerlab and spec.definition.containerlabRef are set,"+
				" exactly one must be provided",
			claberneteserrors.ErrReconcile,
		)
	}

	raw, err := c.loadDefinitionRef(ctx, topology.GetNamespace(), ref)
	if err != nil {
		return nil, err
	}

	working := topology.DeepCopy()
	working.Spec.Definition.Containerlab = raw

	return working, nil
}

// loadDefinitionRef fetches the raw definition referenced by ref -- from a ConfigMap in the given
// namespace or from a URL.
func (c *Controller) loadDefinitionRef(
	ctx context.Context,
	namespace string,
	ref *clabernetesapisv1alpha1.DefinitionRef,
) (string, error) {
	switch {
	case ref.URL != "":
		buf := bytes.NewBuffer(nil)

		err := clabernetesutil.WriteHTTPContentsFromPath(ctx, ref.URL, buf, nil)
		if err != nil {
			return "", fmt.Errorf(
				"%w: failed loading definition from url '%s': %w",
				claberneteserrors.ErrReconcile,
				ref.URL,
				err,
			)
		}

		return buf.String(), nil
	case ref.ConfigMapName != "":
		key := ref.ConfigMapKey
		if key == "" {
			key = defaultDefinitionRefConfigMapKey
		}

		configMap := &k8scorev1.ConfigMap{}

		err := c.BaseController.Client.Get(
			ctx,
			apimachinerytypes.NamespacedName{Namespace: namespace, Name: ref.ConfigMapName},
			configMap,
		)
		if err != nil {
			return "", fmt.Errorf(
				"%w: failed getting definition configmap '%s/%s': %w",
				claberneteserrors.ErrReconcile,
				namespace,
				ref.ConfigMapName,
				err,
			)
		}

		raw, ok := configMap.Data[key]
		if !ok {
			return "", fmt.Errorf(
				"%w: definition configmap '%s/%s' has no key '%s'",
				claberneteserrors.ErrReconcile,
				namespace,
				ref.ConfigMapName,
				key,
			)
		}

		return raw, nil
	default:
		return "", fmt.Errorf(
			"%w: spec.definition.containerlabRef sets neither configMapName nor url",
			claberneteserrors.ErrReconcile,
		)
	}
}
