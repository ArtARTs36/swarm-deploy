package swarm

import "github.com/docker/docker/client"

type Swarm struct {
	ServiceManager *ServiceManager
}

func NewSwarm(dockerClient *client.Client) *Swarm {
	return &Swarm{
		ServiceManager: newServiceManager(dockerClient),
	}
}
