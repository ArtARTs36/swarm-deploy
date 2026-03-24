package tools

import (
	"encoding/json"
	"fmt"

	"github.com/artarts36/swarm-deploy/internal/assistant"
	"github.com/artarts36/swarm-deploy/internal/controller"
)

// Sync triggers manual synchronization run.
type Sync struct {
	control SyncTrigger
}

// NewSync creates sync component.
func NewSync(control SyncTrigger) *Sync {
	return &Sync{control: control}
}

// Definition returns tool metadata visible to the model.
func (s *Sync) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "sync",
		Description: "Triggers manual synchronization run.",
		ParametersJSONSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

// Execute runs sync tool.
func (s *Sync) Execute(_ map[string]any) (string, error) {
	queued := s.control.Trigger(controller.TriggerManual)
	payload := struct {
		Queued bool `json:"queued"`
	}{
		Queued: queued,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode sync tool response: %w", err)
	}

	return string(encoded), nil
}
