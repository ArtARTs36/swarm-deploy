package webserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/artarts36/go-entrypoint"
	"github.com/artarts36/swarm-deploy/internal/controller"
	generated "github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/generated"
	"github.com/artarts36/swarm-deploy/ui"
)

const readHeaderTimeout = 10 * time.Second

type Application struct {
	server *http.Server
}

func NewApplication(address string, control *controller.Controller) (*Application, error) {
	h := NewHandler(control)

	apiHandler, err := generated.NewServer(h)
	if err != nil {
		return nil, fmt.Errorf("build ogen api server: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)

	uiHandler := http.FileServer(http.FS(ui.FS))
	mux.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiHandler))
	mux.Handle("/", uiHandler)

	return &Application{
		server: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}, nil
}

func (a *Application) Entrypoint() entrypoint.Entrypoint {
	return entrypoint.HTTPServer("web-server", a.server)
}
