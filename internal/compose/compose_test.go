package compose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadInitJobsEnvironment(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yaml")
	composePayload := []byte(`
services:
  api:
    image: ghcr.io/acme/api:1.0.0
    x-init-deploy-jobs:
      - name: migrate
        image: ghcr.io/acme/api:1.0.0
        command: ["./bin/migrate", "up"]
        environment:
          DB_HOST: postgres
          DB_USER: app
`)
	if err := os.WriteFile(composePath, composePayload, 0o600); err != nil {
		require.NoError(t, err, "write compose file")
	}

	file, err := Load(composePath)
	require.NoError(t, err, "load compose")

	require.Len(t, file.Services, 1, "expected 1 service")
	require.Len(t, file.Services[0].InitJobs, 1, "expected 1 init job")

	first := file.Services[0].InitJobs[0]
	assert.Equal(t, "postgres", first.Environment["DB_HOST"], "unexpected DB_HOST")
	assert.Equal(t, "app", first.Environment["DB_USER"], "unexpected DB_USER")
}

func TestLoadInitJobsIgnoresLegacyEnv(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yaml")
	composePayload := []byte(`
services:
  api:
    image: ghcr.io/acme/api:1.0.0
    x-init-deploy-jobs:
      - name: legacy
        image: ghcr.io/acme/api:1.0.0
        env:
          LEGACY: "1"
`)
	if err := os.WriteFile(composePath, composePayload, 0o600); err != nil {
		require.NoError(t, err, "write compose file")
	}

	file, err := Load(composePath)
	require.NoError(t, err, "load compose")

	require.Len(t, file.Services, 1, "unexpected services count")
	require.Len(t, file.Services[0].InitJobs, 1, "unexpected init jobs count")
	assert.Nil(t, file.Services[0].InitJobs[0].Environment, "legacy env must be ignored")
}
