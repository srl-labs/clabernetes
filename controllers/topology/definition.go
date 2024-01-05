package topology

import (
	"fmt"

	clabernetesapis "github.com/srl-labs/clabernetes/apis"
	clabernetesapisv1alpha1 "github.com/srl-labs/clabernetes/apis/v1alpha1"
	claberneteserrors "github.com/srl-labs/clabernetes/errors"
	clabernetesutilcontainerlab "github.com/srl-labs/clabernetes/util/containerlab"
	clabernetesutilkne "github.com/srl-labs/clabernetes/util/kne"
	"gopkg.in/yaml.v3"
)

func (c *Controller) processDefinition(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	switch {
	case topology.Spec.Definition.Containerlab != "":
		return c.processContainerlabDefinition(topology, reconcileData)
	case topology.Spec.Definition.Kne != "":
		return c.processKneDefinition(topology, reconcileData)
	default:
		return fmt.Errorf(
			"%w: unknown or unsupported topology definition knid, this is *probably* a bug",
			claberneteserrors.ErrReconcile,
		)
	}
}

func (c *Controller) processContainerlabDefinition(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	reconcileData.Kind = clabernetesapis.TopologyKindContainerlab

	// load the containerlab topo from the CR to make sure its all good
	containerlabConfig, err := clabernetesutilcontainerlab.LoadContainerlabConfig(
		topology.Spec.Definition.Containerlab,
	)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing containerlab config, error: %s", err)

		return err
	}

	// we may have *different defaults per "sub-topology" so we do a cheater "deep copy" by just
	// marshalling here and unmarshalling per node in the process func :)
	defaultsYAML, err := yaml.Marshal(containerlabConfig.Topology.Defaults)
	if err != nil {
		return err
	}

	for nodeName := range containerlabConfig.Topology.Nodes {
		err = processConfigForNode(
			c.Log,
			topology,
			containerlabConfig,
			nodeName,
			defaultsYAML,
			reconcileData,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) processKneDefinition(
	topology *clabernetesapisv1alpha1.Topology,
	reconcileData *ReconcileData,
) error {
	reconcileData.Kind = clabernetesapis.TopologyKindKne

	// load the kne topo to make sure its all good
	kneTopo, err := clabernetesutilkne.LoadKneTopology(topology.Spec.Definition.Kne)
	if err != nil {
		c.BaseController.Log.Criticalf("failed parsing kne topology, error: %s", err)

		return err
	}

	return processKneDefinition(c.Log, topology, kneTopo, reconcileData)
}
