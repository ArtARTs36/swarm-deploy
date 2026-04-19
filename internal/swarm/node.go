package swarm

// Node is a persisted/read model of Docker Swarm node.
type Node struct {
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
