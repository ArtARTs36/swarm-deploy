package webserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-deploy/swarm-deploy/internal/config"
)

func TestUIRoutes(t *testing.T) {
	app, err := NewApplication(":0", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, config.AuthenticationSpec{})
	require.NoError(t, err, "new application")

	testCases := []struct {
		name           string
		path           string
		wantCode       int
		wantLocation   string
		locationAssert bool
	}{
		{
			name:     "root serves spa index",
			path:     "/",
			wantCode: 200,
		},
		{
			name:     "overview route uses spa fallback",
			path:     "/overview",
			wantCode: 200,
		},
		{
			name:     "secrets route uses spa fallback",
			path:     "/secrets",
			wantCode: 200,
		},
		{
			name:           "ui root redirects to overview",
			path:           "/ui",
			wantCode:       301,
			wantLocation:   "/overview",
			locationAssert: true,
		},
		{
			name:           "ui prefix redirects to overview",
			path:           "/ui/legacy",
			wantCode:       301,
			wantLocation:   "/overview",
			locationAssert: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			app.server.Handler.ServeHTTP(rec, req)

			assert.Equal(t, testCase.wantCode, rec.Code, "status mismatch")
			if testCase.locationAssert {
				assert.Equal(t, testCase.wantLocation, rec.Header().Get("Location"), "redirect mismatch")
			}
		})
	}
}

func TestPasskeyRoutesMountedAndPublic(t *testing.T) {
	authCfg := config.AuthenticationSpec{
		Passkey: config.PasskeyAuthenticationSpec{
			Enabled:        true,
			RPID:           "localhost",
			RPDisplayName:  "Swarm Deploy",
			RPOrigins:      []string{"http://localhost:8080"},
			StoragePath:    t.TempDir(),
			InsecureCookie: true,
		},
	}

	app, err := NewApplication(":0", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, authCfg)
	require.NoError(t, err, "new application")

	req := httptest.NewRequest(http.MethodPost, "/api/passkey/registerBegin", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "passkey route should be reachable without prior authentication")
}

func TestAuthMethodsRoutePublic(t *testing.T) {
	authCfg := config.AuthenticationSpec{
		Passkey: config.PasskeyAuthenticationSpec{
			Enabled:        true,
			RPID:           "localhost",
			RPDisplayName:  "Swarm Deploy",
			RPOrigins:      []string{"http://localhost:8080"},
			StoragePath:    t.TempDir(),
			InsecureCookie: true,
		},
	}

	app, err := NewApplication(":0", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, authCfg)
	require.NoError(t, err, "new application")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/methods", nil)
	rec := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "auth methods route should be public")

	var payload map[string]bool
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload), "decode auth methods payload")
	assert.Equal(t, false, payload["basic_enabled"], "unexpected basic_enabled")
	assert.Equal(t, true, payload["passkey_enabled"], "unexpected passkey_enabled")
}
