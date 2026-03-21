package middlewares

import (
	"log/slog"
	"net/http"

	"github.com/artarts36/swarm-deploy/internal/entrypoints/webserver/authenticator"
	"github.com/artarts36/swarm-deploy/internal/event/dispatcher"
	"github.com/artarts36/swarm-deploy/internal/event/events"
)

const (
	authSessionCookieName  = "swarm_deploy_auth_session"
	authSessionCookieValue = "1"
)

func Authorize(
	next http.Handler,
	auth authenticator.Authenticator,
	eventDispatcher dispatcher.Dispatcher,
) http.Handler {
	if auth == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "[security] authorizing request")

		if auth.Authenticate(r) {
			slog.InfoContext(r.Context(), "[security] request authorized")
			username, _, _ := r.BasicAuth()
			if !hasActiveAuthSession(r) {
				eventDispatcher.Dispatch(r.Context(), &events.UserAuthenticated{Username: username})
				setActiveAuthSession(w)
			}
			next.ServeHTTP(w, r)
			return
		}

		slog.InfoContext(r.Context(), "[security] request rejected")
		auth.Challenge(w)
	})
}

func hasActiveAuthSession(r *http.Request) bool {
	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil {
		return false
	}

	return cookie.Value == authSessionCookieValue
}

func setActiveAuthSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    authSessionCookieValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
