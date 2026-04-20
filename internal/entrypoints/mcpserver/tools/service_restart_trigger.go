package tools

import (
	"context"
	"fmt"

	"github.com/swarm-deploy/swarm-deploy/internal/entrypoints/mcpserver/routing"
	"github.com/swarm-deploy/swarm-deploy/internal/event/dispatcher"
	"github.com/swarm-deploy/swarm-deploy/internal/event/events"
)

// RestartService restarts stack service by scaling replicas to zero and restoring previous value.
type RestartService struct {
	manager         ServiceReplicasManager
	eventDispatcher dispatcher.Dispatcher
}

// NewRestartService creates service_restart_trigger component.
func NewRestartService(manager ServiceReplicasManager, eventDispatcher dispatcher.Dispatcher) *RestartService {
	return &RestartService{
		manager:         manager,
		eventDispatcher: eventDispatcher,
	}
}

// Definition returns tool metadata visible to the model.
func (s *RestartService) Definition() routing.ToolDefinition {
	return routing.ToolDefinition{
		Name:        "service_restart_trigger",
		Description: "Restarts a stack service by scaling replicas to zero and restoring previous count.",
		ParametersJSONSchema: map[string]any{
			"type": "object",
			"required": []string{
				"stack",
				"service",
			},
			"properties": map[string]any{
				"stack": map[string]any{
					"type":        "string",
					"description": "Stack name.",
				},
				"service": map[string]any{
					"type":        "string",
					"description": "Service name inside stack.",
				},
			},
		},
	}
}

// Execute runs service_restart_trigger tool.
func (s *RestartService) Execute(ctx context.Context, request routing.Request) (routing.Response, error) {
	target, err := parseServiceReplicasTarget(request.Payload)
	if err != nil {
		return routing.Response{}, err
	}

	replicas, err := s.manager.Restart(ctx, target)
	if err != nil {
		return routing.Response{}, fmt.Errorf("restart service: %w", err)
	}

	s.eventDispatcher.Dispatch(ctx, &events.ServiceRestarted{
		StackName:   target.StackName(),
		ServiceName: target.ServiceName(),
	})

	payload := struct {
		Stack    string `json:"stack"`
		Service  string `json:"service"`
		Replicas uint64 `json:"replicas"`
	}{
		Stack:    target.StackName(),
		Service:  target.ServiceName(),
		Replicas: replicas,
	}

	return routing.Response{
		Payload: payload,
	}, nil
}
