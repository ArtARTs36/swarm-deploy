package tools

import (
	"encoding/json"
	"fmt"

	"github.com/artarts36/swarm-deploy/internal/assistant"
	"github.com/artarts36/swarm-deploy/internal/swarm"
)

// ListNodes returns current Docker Swarm nodes snapshot.
type ListNodes struct {
	nodes NodesReader
}

// NewListNodes creates list_nodes component.
func NewListNodes(nodesStore NodesReader) *ListNodes {
	return &ListNodes{nodes: nodesStore}
}

// Definition returns tool metadata visible to the model.
func (l *ListNodes) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "list_nodes",
		Description: "Returns current Docker Swarm nodes snapshot.",
		ParametersJSONSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// Execute runs list_nodes tool.
func (l *ListNodes) Execute(_ map[string]any) (string, error) {
	if l.nodes == nil {
		return "", fmt.Errorf("nodes store is not configured")
	}

	payload := struct {
		Nodes []swarm.NodeInfo `json:"nodes"`
	}{
		Nodes: l.nodes.List(),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode nodes tool response: %w", err)
	}

	return string(encoded), nil
}
