package swarm

import "github.com/docker/docker/client"

type Swarm struct {
	Services *ServiceManager
	Secrets  *SecretManager
	Nodes    *NodeManager
	Networks *NetworkManager
	Plugins  *PluginManager
}

func NewSwarm(dockerClient *client.Client) *Swarm {
	return &Swarm{
		Services: newServiceManager(dockerClient),
		Secrets:  newSecretManager(dockerClient),
		Nodes:    newNodeManager(dockerClient),
		Networks: newNetworkManager(dockerClient),
		Plugins:  newPluginManager(dockerClient),
	}
}
