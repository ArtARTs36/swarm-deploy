package middlewares

import (
	"log/slog"
	"net/http"

	"github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/authenticator"
)

func Authorize(next http.Handler, auth authenticator.Authenticator) http.Handler {
	if auth == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "[security] authorizing request")

		if auth.Authenticate(r) {
			slog.InfoContext(r.Context(), "[security] request authorized")
			next.ServeHTTP(w, r)
			return
		}

		slog.InfoContext(r.Context(), "[security] request rejected")
		auth.Challenge(w)
	})
}
