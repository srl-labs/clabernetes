package kne

import (
	"fmt"

	featureprofilestopologybinding "github.com/openconfig/featureprofiles/topologies/proto/binding"
	knetopologyproto "github.com/openconfig/kne/proto/topo"
	ondatraproto "github.com/openconfig/ondatra/proto"
	clabernetesapistopologyv1alpha1 "github.com/srl-labs/clabernetes/apis/topology/v1alpha1"
)

func (c *Controller) reconcileFeatureProfilesTopo(
	kne *clabernetesapistopologyv1alpha1.Kne,
	kneTopo *knetopologyproto.Topology,
) bool {
	binding := renderTopoBinding(kne, kneTopo)

	testbed := renderTopoTestbed(kneTopo)

	var shouldUpdate bool

	bindingString := binding.String()

	if bindingString != kne.Status.FeatureProfilesBinding {
		shouldUpdate = true

		kne.Status.FeatureProfilesBinding = bindingString
	}

	testbedString := testbed.String()

	if testbedString != kne.Status.FeatureProfilesTestbed {
		shouldUpdate = true

		kne.Status.FeatureProfilesTestbed = testbedString
	}

	return shouldUpdate
}

func renderTopoBinding(
	kne *clabernetesapistopologyv1alpha1.Kne,
	kneTopo *knetopologyproto.Topology,
) *featureprofilestopologybinding.Binding {
	binding := &featureprofilestopologybinding.Binding{
		Duts:    make([]*featureprofilestopologybinding.Device, len(kneTopo.Nodes)),
		Ates:    make([]*featureprofilestopologybinding.Device, 0),
		Options: &featureprofilestopologybinding.Options{},
	}

	for idx := range kneTopo.Nodes {
		var lbAddress string

		nodeStatus, ok := kne.Status.NodeExposedPorts[kneTopo.Nodes[idx].Name]
		if ok {
			lbAddress = nodeStatus.LoadBalancerAddress
		}

		ports := make([]*featureprofilestopologybinding.Port, len(kneTopo.Nodes[idx].Interfaces))

		var portIdx int

		for portKeyName := range kneTopo.Nodes[idx].Interfaces {
			ports[portIdx] = &featureprofilestopologybinding.Port{
				Id:   portKeyName,
				Name: kneTopo.Nodes[idx].Interfaces[portKeyName].Name,
			}

			portIdx++
		}

		binding.Duts[idx] = &featureprofilestopologybinding.Device{
			Id:   kneTopo.Nodes[idx].Name,
			Name: kneTopo.Nodes[idx].Name,
			Options: &featureprofilestopologybinding.Options{
				Target: lbAddress,
				// TODO obviously we'll need to either leave this out or have some spec field to
				//  set it or something.
				Username: "admin",
				Password: "NokiaSrl1!",
			},
			Ports: ports,
			Ssh: &featureprofilestopologybinding.Options{
				Target: fmt.Sprintf("%s:22", lbAddress),
			},
			Gnmi: &featureprofilestopologybinding.Options{
				Target:     fmt.Sprintf("%s:57400", lbAddress),
				SkipVerify: true,
			},
			Gnoi: &featureprofilestopologybinding.Options{
				Target:     fmt.Sprintf("%s:57400", lbAddress),
				SkipVerify: true,
			},
			Gnsi: &featureprofilestopologybinding.Options{
				Target:     fmt.Sprintf("%s:57400", lbAddress),
				SkipVerify: true,
			},
			Gribi: &featureprofilestopologybinding.Options{
				Target:     fmt.Sprintf("%s:57401", lbAddress),
				SkipVerify: true,
			},
			Vendor: ondatraproto.Device_Vendor(
				ondatraproto.Device_Vendor_value[kneTopo.Nodes[idx].Vendor.String()],
			),
			HardwareModel:   kneTopo.Nodes[idx].Model,
			SoftwareVersion: kneTopo.Nodes[idx].Version,
		}
	}

	return binding
}

func renderTopoTestbed(
	kneTopo *knetopologyproto.Topology,
) *ondatraproto.Testbed {
	testbed := &ondatraproto.Testbed{
		Duts:  make([]*ondatraproto.Device, len(kneTopo.Nodes)),
		Ates:  make([]*ondatraproto.Device, 0),
		Links: make([]*ondatraproto.Link, len(kneTopo.Links)),
	}

	for idx := range kneTopo.Nodes {
		ports := make([]*ondatraproto.Port, len(kneTopo.Nodes[idx].Interfaces))

		var portIdx int

		for portKeyName := range kneTopo.Nodes[idx].Interfaces {
			ports[portIdx] = &ondatraproto.Port{
				Id: portKeyName,
			}

			portIdx++
		}

		testbed.Duts[idx] = &ondatraproto.Device{
			Id: kneTopo.Nodes[idx].Name,
			Vendor: ondatraproto.Device_Vendor(
				ondatraproto.Device_Vendor_value[kneTopo.Nodes[idx].Vendor.String()],
			),
			Ports: ports,
		}
	}

	for idx := range kneTopo.Links {
		testbed.Links[idx] = &ondatraproto.Link{
			A: fmt.Sprintf("%s:%s", kneTopo.Links[idx].ANode, kneTopo.Links[idx].AInt),
			B: fmt.Sprintf("%s:%s", kneTopo.Links[idx].ZNode, kneTopo.Links[idx].ZInt),
		}
	}

	return testbed
}
