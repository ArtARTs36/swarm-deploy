package swarm

import "github.com/docker/docker/client"

type Swarm struct {
	Services     *ServiceManager
	Secrets      *SecretManager
	Nodes        *NodeManager
	Networks     *NetworkManager
	Plugins      *PluginManager
	BinaryRunner *BinaryRunner
}

func NewSwarm(dockerClient *client.Client, command string) *Swarm {
	return &Swarm{
		Services:     newServiceManager(dockerClient),
		Secrets:      newSecretManager(dockerClient),
		Nodes:        newNodeManager(dockerClient),
		Networks:     newNetworkManager(dockerClient),
		Plugins:      newPluginManager(dockerClient),
		BinaryRunner: newBinaryRunner(command),
	}
}
