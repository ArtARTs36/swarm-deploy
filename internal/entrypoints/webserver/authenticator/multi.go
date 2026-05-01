package authenticator

import (
	"net/http"

	"github.com/swarm-deploy/swarm-deploy/internal/security"
)

type multiAuthenticator struct {
	authenticators []Authenticator
}

func (a *multiAuthenticator) Authenticate(r *http.Request) (security.User, bool) {
	for _, auth := range a.authenticators {
		user, ok := auth.Authenticate(r)
		if ok {
			return user, true
		}
	}

	return security.User{}, false
}

func (a *multiAuthenticator) Challenge(w http.ResponseWriter) {
	if len(a.authenticators) == 0 {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	a.authenticators[0].Challenge(w)
}

func (a *multiAuthenticator) MountAPIRoutes(mux *http.ServeMux) {
	for _, auth := range a.authenticators {
		mounter, ok := auth.(APIRoutesMounter)
		if ok {
			mounter.MountAPIRoutes(mux)
		}
	}
}

func (a *multiAuthenticator) IsPublicPath(path string) bool {
	for _, auth := range a.authenticators {
		matcher, ok := auth.(PublicPathMatcher)
		if ok && matcher.IsPublicPath(path) {
			return true
		}
	}

	return false
}
