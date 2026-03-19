package apiserver

import (
	"context"

	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/apiserver/generated"
)

func (h *handler) ListStacks(_ context.Context) (*generated.StacksResponse, error) {
	return &generated.StacksResponse{
		Stacks: toGeneratedStacks(h.control.ListStacks()),
		Sync:   h.control.LastSyncInfo(),
	}, nil
}
