package topology

import (
	"sort"

	clabernetesapistopologyv1alpha1 "gitlab.com/carlmontanari/clabernetes/apis/topology/v1alpha1"
)

// AllocateTunnelIDs processes the given tunnels and allocates vnids. This function updates the
// given status object by iterating over the freshly processed tunnels (as processed during a
// reconciliation) and assigning any tunnels in the status without a vnid the next valid vnid.
func AllocateTunnelIDs(
	statusTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
	processedTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) {
	// we want to allocate ids deterministically, so lets iterate over the maps in *order* by
	// getting a sorted list of keys and then iterating over those
	processedTunnelsSortedKeys := make([]string, len(processedTunnels))

	var processedIdx int

	for k := range processedTunnels {
		processedTunnelsSortedKeys[processedIdx] = k

		processedIdx++
	}

	sort.Strings(processedTunnelsSortedKeys)

	// iterate over stored (in status) tunnels and allocate previously assigned ids to any relevant
	// tunnels -- while doing so, make a map of all allocated tunnel ids so we can make sure to not
	// re-use things.
	allocatedTunnelIds := make(map[int]bool)

	for nodeName, nodeTunnels := range processedTunnels {
		existingNodeTunnels, ok := statusTunnels[nodeName]
		if !ok {
			continue
		}

		for _, newTunnel := range nodeTunnels {
			for _, existingTunnel := range existingNodeTunnels {
				if newTunnel.LocalLinkName == existingTunnel.LocalLinkName &&
					newTunnel.RemoteName == existingTunnel.RemoteName {
					newTunnel.ID = existingTunnel.ID

					allocatedTunnelIds[newTunnel.ID] = true

					break
				}
			}
		}
	}

	for _, nodeName := range processedTunnelsSortedKeys {
		nodeTunnels := processedTunnels[nodeName]

		for _, tunnel := range nodeTunnels {
			if tunnel.ID != 0 {
				continue
			}

			// iterate over the tunnels to see if this tunnels remote pair already has a vnid set,
			// if *yes* we need to re-use that vnid obviously!
			idToAssign := findAllocatedIDIfExists(
				nodeName,
				tunnel.RemoteNodeName,
				tunnel.LocalLinkName,
				processedTunnelsSortedKeys,
				processedTunnels,
			)

			if idToAssign != 0 {
				tunnel.ID = idToAssign
				allocatedTunnelIds[idToAssign] = true

				continue
			}

			// dear lord this is probably ridiculous but should be fine for now... :)
			for i := 1; i < 16_000_000; i++ {
				_, ok := allocatedTunnelIds[i]
				if ok {
					// already allocated
					continue
				}

				tunnel.ID = i
				allocatedTunnelIds[i] = true

				break
			}
		}
	}
}

func findAllocatedIDIfExists(
	nodeName, tunnelRemoteNodeName, tunnelLocalLinkName string,
	sortedKeys []string,
	processedTunnels map[string][]*clabernetesapistopologyv1alpha1.Tunnel,
) int {
	for _, remoteNodeName := range sortedKeys {
		if nodeName == remoteNodeName {
			// this is us, next...
			continue
		}

		if tunnelRemoteNodeName != remoteNodeName {
			// this remote node name doesnt match our current tunnels remote node name
			// so not our remote end, next...
			continue
		}

		for _, remoteTunnel := range processedTunnels[remoteNodeName] {
			if remoteTunnel.RemoteNodeName != nodeName {
				// tunnel not between this node pair
				continue
			}

			if tunnelLocalLinkName != remoteTunnel.RemoteLinkName {
				// this specific tunnel does not match our local tunnel
				continue
			}

			if remoteTunnel.ID == 0 {
				// we found our remote tunnel but vnid is not set yet, so we'll just keep
				// doing our thing
				return 0
			}

			return remoteTunnel.ID
		}
	}

	return 0
}
