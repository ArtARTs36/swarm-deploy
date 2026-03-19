package apiserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/artarts36/go-entrypoint"
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/apiserver/generated"
)

const readHeaderTimeout = 10 * time.Second

type Application struct {
	server *http.Server
}

func NewApplication(address string, control *controller.Controller) (*Application, error) {
	h := NewHandler(control)

	server, err := generated.NewServer(h)
	if err != nil {
		return nil, fmt.Errorf("build ogen api server: %w", err)
	}

	return &Application{
		server: &http.Server{
			Addr:              address,
			Handler:           server,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}, nil
}

func (a *Application) Entrypoint() entrypoint.Entrypoint {
	return entrypoint.HTTPServer("api-server", a.server)
}
