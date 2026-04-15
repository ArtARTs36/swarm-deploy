package tools

import (
	"context"
	"fmt"

	"github.com/artarts36/swarm-deploy/internal/entrypoints/mcpserver/routing"
	"github.com/artarts36/swarm-deploy/internal/swarm/inspector"
)

// DockerPluginList returns current Docker plugins snapshot.
type DockerPluginList struct {
	inspector PluginInspector
}

// NewDockerPluginList creates docker_plugin_list component.
func NewDockerPluginList(pluginInspector PluginInspector) *DockerPluginList {
	return &DockerPluginList{
		inspector: pluginInspector,
	}
}

// Definition returns tool metadata visible to the model.
func (l *DockerPluginList) Definition() routing.ToolDefinition {
	return routing.ToolDefinition{
		Name:        "docker_plugin_list",
		Description: "Returns current Docker plugins snapshot.",
		ParametersJSONSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// Execute runs docker_plugin_list tool.
func (l *DockerPluginList) Execute(ctx context.Context, _ routing.Request) (routing.Response, error) {
	plugins, err := l.inspector.InspectPlugins(ctx)
	if err != nil {
		return routing.Response{}, fmt.Errorf("inspect plugins: %w", err)
	}

	payload := struct {
		Plugins []inspector.PluginInfo `json:"plugins"`
	}{
		Plugins: plugins,
	}

	return routing.Response{
		Payload: payload,
	}, nil
}
