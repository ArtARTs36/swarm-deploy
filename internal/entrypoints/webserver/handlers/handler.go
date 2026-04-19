package handlers

import (
	"context"

	"github.com/artarts36/swarm-deploy/internal/assistant"
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/generated"
	"github.com/artarts36/swarm-deploy/internal/event/history"
	swarmnode "github.com/artarts36/swarm-deploy/internal/node"
	"github.com/artarts36/swarm-deploy/internal/service"
	"github.com/artarts36/swarm-deploy/internal/swarm"
)

// ServiceStatusInspector reads compact status snapshot for a stack service.
type ServiceStatusInspector interface {
	// InspectServiceStatus returns compact status snapshot for a stack service.
	GetStatus(ctx context.Context, serviceRef swarm.ServiceReference) (swarm.ServiceStatus, error)
}

type handler struct {
	generated.UnimplementedHandler
	control          *controller.Controller
	serviceInspector ServiceStatusInspector
	history          *history.Store
	services         *service.Store
	nodes            *swarmnode.Store
	assistant        assistant.Assistant
}

var _ generated.Handler = (*handler)(nil)

func New(
	control *controller.Controller,
	serviceInspector ServiceStatusInspector,
	history *history.Store,
	services *service.Store,
	nodes *swarmnode.Store,
	assistantService assistant.Assistant,
) generated.Handler {
	return &handler{
		control:          control,
		serviceInspector: serviceInspector,
		history:          history,
		services:         services,
		nodes:            nodes,
		assistant:        assistantService,
	}
}
