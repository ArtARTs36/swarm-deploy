package authenticator

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/artarts36/specw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-deploy/swarm-deploy/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestCreatePasskeyAuthenticator(t *testing.T) {
	auth, err := Create(config.AuthenticationSpec{
		Passkey: config.PasskeyAuthenticationSpec{
			Enabled:        true,
			RPID:           "localhost",
			RPDisplayName:  "Swarm Deploy",
			RPOrigins:      []string{"http://localhost:8080"},
			StoragePath:    t.TempDir(),
			InsecureCookie: true,
		},
	})
	require.NoError(t, err, "create authenticator")
	require.NotNil(t, auth, "expected passkey authenticator")

	_, supportsRoutes := auth.(APIRoutesMounter)
	assert.True(t, supportsRoutes, "expected APIRoutesMounter")

	pathMatcher, supportsPublicPaths := auth.(PublicPathMatcher)
	require.True(t, supportsPublicPaths, "expected PublicPathMatcher")
	assert.True(t, pathMatcher.IsPublicPath("/api/passkey/loginBegin"), "expected public passkey path")
}

func TestCreateBasicAndPasskeyAuthenticatorPreservesBasicAuth(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err, "generate bcrypt hash")

	auth, err := Create(config.AuthenticationSpec{
		Basic: config.BasicAuthenticationSpec{
			HTPasswdFile: specw.File{
				Path:    "/run/secrets/basic.htpasswd",
				Content: []byte("admin:" + string(hash) + "\n"),
			},
		},
		Passkey: config.PasskeyAuthenticationSpec{
			Enabled:        true,
			RPID:           "localhost",
			RPDisplayName:  "Swarm Deploy",
			RPOrigins:      []string{"http://localhost:8080"},
			StoragePath:    t.TempDir(),
			InsecureCookie: true,
		},
	})
	require.NoError(t, err, "create authenticator")
	require.NotNil(t, auth, "expected composite authenticator")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stacks", nil)
	req.SetBasicAuth("admin", "secret")

	user, authenticated := auth.Authenticate(req)
	require.True(t, authenticated, "expected basic auth success")
	assert.Equal(t, "admin", user.Name, "unexpected authenticated user")

	rec := httptest.NewRecorder()
	auth.Challenge(rec)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "unexpected challenge status")
	assert.Contains(
		t,
		rec.Header().Get("WWW-Authenticate"),
		`Basic realm="swarm-deploy"`,
		"expected basic challenge header",
	)
}
