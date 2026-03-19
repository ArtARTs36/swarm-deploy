package apiserver

import (
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/apiserver/generated"
)

type handler struct {
	generated.UnimplementedHandler
	control *controller.Controller
}

var _ generated.Handler = (*handler)(nil)

func NewHandler(control *controller.Controller) generated.Handler {
	return &handler{
		control: control,
	}
}
