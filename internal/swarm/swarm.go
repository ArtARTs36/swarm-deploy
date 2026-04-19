package swarm

import "github.com/docker/docker/client"

type Swarm struct {
	Services *ServiceManager
	Secrets  *SecretManager
}

func NewSwarm(dockerClient *client.Client) *Swarm {
	return &Swarm{
		Services: newServiceManager(dockerClient),
		Secrets:  newSecretManager(dockerClient),
	}
}
