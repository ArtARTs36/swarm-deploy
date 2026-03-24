package tools

import "github.com/artarts36/swarm-deploy/internal/assistant"

// Tool describes one MCP tool implementation.
type Tool interface {
	// Definition returns metadata visible to the model.
	Definition() assistant.ToolDefinition
	// Execute runs tool logic with decoded JSON arguments.
	Execute(arguments map[string]any) (string, error)
}
