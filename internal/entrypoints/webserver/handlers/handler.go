package handlers

import (
	"github.com/artarts36/swarm-deploy/internal/assistant"
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/generated"
	"github.com/artarts36/swarm-deploy/internal/event/history"
	"github.com/artarts36/swarm-deploy/internal/service"
	swarminspector "github.com/artarts36/swarm-deploy/internal/swarm/inspector"
)

type handler struct {
	generated.UnimplementedHandler
	control      *controller.Controller
	inspectorSvc *swarminspector.Inspector
	history      *history.Store
	services     *service.Store
	nodes        *swarminspector.NodeStore
	assistant    assistant.Assistant
}

var _ generated.Handler = (*handler)(nil)

func New(
	control *controller.Controller,
	inspectorSvc *swarminspector.Inspector,
	history *history.Store,
	services *service.Store,
	nodes *swarminspector.NodeStore,
	assistantService assistant.Assistant,
) generated.Handler {
	return &handler{
		control:      control,
		inspectorSvc: inspectorSvc,
		history:      history,
		services:     services,
		nodes:        nodes,
		assistant:    assistantService,
	}
}
