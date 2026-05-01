package authenticator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"

	passkeylib "github.com/egregors/passkey"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/swarm-deploy/swarm-deploy/internal/config"
	"github.com/swarm-deploy/swarm-deploy/internal/security"
)

const (
	passkeyRoutePrefix = "/api/passkey/"
	passkeyUserKey     = passkeylib.AuthUserIDKey("swarmDeployPasskeyUserID")
)

type passkeyAuthenticator struct {
	passkey   *passkeylib.Passkey
	userStore *passkeyUserStore
	withAuth  func(next http.Handler) http.Handler
}

func newPasskeyAuthenticator(cfg config.PasskeyAuthenticationSpec) (Authenticator, error) {
	userStore, err := newPasskeyUserStore(filepath.Join(cfg.StoragePath, "users.json"))
	if err != nil {
		return nil, err
	}

	authSessionStore, err := newPasskeySessionStore[webauthn.SessionData](
		filepath.Join(cfg.StoragePath, "auth-sessions.json"),
	)
	if err != nil {
		return nil, err
	}

	userSessionStore, err := newPasskeySessionStore[passkeylib.UserSessionData](
		filepath.Join(cfg.StoragePath, "user-sessions.json"),
	)
	if err != nil {
		return nil, err
	}

	options := []passkeylib.Option{
		passkeylib.WithSessionCookieNamePrefix("swarmDeploy"),
	}
	if cfg.InsecureCookie {
		options = append(options, passkeylib.WithInsecureCookie())
	}

	passkeyService, err := passkeylib.New(
		passkeylib.Config{
			WebauthnConfig: &webauthn.Config{
				RPID:          cfg.RPID,
				RPDisplayName: cfg.RPDisplayName,
				RPOrigins:     cfg.RPOrigins,
			},
			UserStore:        userStore,
			AuthSessionStore: authSessionStore,
			UserSessionStore: userSessionStore,
		},
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("create passkey service: %w", err)
	}

	return &passkeyAuthenticator{
		passkey:   passkeyService,
		userStore: userStore,
		withAuth:  passkeyService.Auth(passkeyUserKey, nil, nil),
	}, nil
}

func (a *passkeyAuthenticator) Authenticate(r *http.Request) (security.User, bool) {
	authenticated := false
	resolvedUser := security.User{}

	handler := a.withAuth(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		userID, ok := passkeylib.UserIDFromCtx(request.Context(), passkeyUserKey)
		if !ok {
			return
		}

		user, err := a.userStore.Get(userID)
		if err != nil {
			return
		}

		authenticated = true
		resolvedUser = security.User{Name: user.WebAuthnName()}
	}))

	handler.ServeHTTP(httptest.NewRecorder(), r)

	return resolvedUser, authenticated
}

func (a *passkeyAuthenticator) Challenge(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (a *passkeyAuthenticator) MountAPIRoutes(mux *http.ServeMux) {
	passkeyMux := http.NewServeMux()
	a.passkey.MountRoutes(passkeyMux, "/api/")

	mux.Handle("/api/passkey/", passkeyMux)
	mux.Handle("/api/passkey", http.RedirectHandler("/api/passkey/", http.StatusMovedPermanently))
}

func (a *passkeyAuthenticator) IsPublicPath(path string) bool {
	return path == strings.TrimSuffix(passkeyRoutePrefix, "/") || strings.HasPrefix(path, passkeyRoutePrefix)
}
