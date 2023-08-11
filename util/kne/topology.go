package kne

import (
	knetopologyproto "github.com/openconfig/kne/proto/topo"
	"google.golang.org/protobuf/encoding/prototext"
)

// LoadKneTopology loads a kne topology definition from a raw kne topology.
func LoadKneTopology(rawTopology string) (*knetopologyproto.Topology, error) {
	topology := &knetopologyproto.Topology{}

	err := prototext.Unmarshal([]byte(rawTopology), topology)
	if err != nil {
		return nil, err
	}

	return topology, nil
}
