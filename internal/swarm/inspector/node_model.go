package inspector

import (
	"sort"
	"strings"
)

// NodeInfo is a persisted/read model of Docker Swarm node.
type NodeInfo struct {
	// ID is a unique Docker Swarm node identifier.
	ID string `json:"id"`
	// Hostname is a node hostname reported by Docker.
	Hostname string `json:"hostname"`
	// Status is a current node runtime state.
	Status string `json:"status"`
	// Availability is a desired scheduling availability.
	Availability string `json:"availability"`
	// ManagerStatus is a manager role/reachability projection.
	ManagerStatus string `json:"manager_status"`
	// EngineVersion is a Docker engine version on node.
	EngineVersion string `json:"engine_version"`
	// Addr is a node address from node status.
	Addr string `json:"addr"`
}

func normalizeNodeInfo(node NodeInfo) NodeInfo {
	node.ID = strings.TrimSpace(node.ID)
	node.Hostname = strings.TrimSpace(node.Hostname)
	node.Status = strings.TrimSpace(node.Status)
	node.Availability = strings.TrimSpace(node.Availability)
	node.ManagerStatus = strings.TrimSpace(node.ManagerStatus)
	node.EngineVersion = strings.TrimSpace(node.EngineVersion)
	node.Addr = strings.TrimSpace(node.Addr)

	return node
}

func sortNodeInfos(nodes []NodeInfo) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Hostname != nodes[j].Hostname {
			return nodes[i].Hostname < nodes[j].Hostname
		}

		return nodes[i].ID < nodes[j].ID
	})
}
