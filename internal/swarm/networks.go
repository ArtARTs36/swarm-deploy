package swarm

import (
	"context"
	"fmt"
	"sort"

	dockernetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// NetworkManager reads current Docker networks snapshot.
type NetworkManager struct {
	dockerClient *client.Client
}

// Network is a runtime snapshot of Docker network metadata.
type Network struct {
	// Name is a Docker network name.
	Name string `json:"name"`
	// Scope describes where network exists (for example: local or swarm).
	Scope string `json:"scope"`
	// Driver is a Docker network driver name.
	Driver string `json:"driver"`
	// Internal indicates that network is internal-only.
	Internal bool `json:"internal"`
	// Attachable indicates network can be attached by standalone containers.
	Attachable bool `json:"attachable"`
	// Ingress indicates swarm routing-mesh ingress network.
	Ingress bool `json:"ingress"`
	// Labels contains custom Docker network labels.
	Labels map[string]string `json:"labels"`
}

func newNetworkManager(dockerClient *client.Client) *NetworkManager {
	return &NetworkManager{
		dockerClient: dockerClient,
	}
}

// List returns current Docker networks snapshot.
func (m *NetworkManager) List(ctx context.Context) ([]Network, error) {
	networks, err := m.dockerClient.NetworkList(ctx, dockernetwork.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list docker networks: %w", err)
	}

	mapped := make([]Network, len(networks))
	for i, network := range networks {
		mapped[i] = m.mapNetwork(network)
	}
	m.sortNetworks(mapped)

	return mapped, nil
}

func (*NetworkManager) mapNetwork(network dockernetwork.Summary) Network {
	return Network{
		Name:       network.Name,
		Scope:      network.Scope,
		Driver:     network.Driver,
		Internal:   network.Internal,
		Attachable: network.Attachable,
		Ingress:    network.Ingress,
		Labels:     network.Labels,
	}
}

func (*NetworkManager) sortNetworks(networks []Network) {
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
}
