package event

import "github.com/artarts36/swarm-deploy/internal/compose"

type SuccessfulDeployEvent struct {
	StackName string
	Commit    string
	Services  []compose.Service
}

type FailedDeployEvent struct {
	StackName string
	Commit    string
	Services  []compose.Service
	Error     error
}
