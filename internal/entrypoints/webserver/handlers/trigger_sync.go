package handlers

import (
	"context"

	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/generated"
)

func (h *handler) TriggerSync(ctx context.Context) (*generated.QueueResponse, error) {
	return &generated.QueueResponse{
		Queued: h.control.Manual(ctx),
	}, nil
}
