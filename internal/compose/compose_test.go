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

func TestLoadResolvesNetworkAliasesToNames(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yaml")
	composePayload := []byte(`
services:
  content-discovery-grpc:
    image: wmb-prod.cr.cloud.ru/services/content-discovery-grpc:latest
    networks:
      - infra
    x-init-deploy-jobs:
      - name: migrations
        image: wmb-prod.cr.cloud.ru/services/content-discovery-migrations:latest
      - name: explicit-network
        image: wmb-prod.cr.cloud.ru/services/content-discovery-migrations:latest
        networks:
          - infra
networks:
  infra:
    name: wmb-infra
    external: true
`)
	if err := os.WriteFile(composePath, composePayload, 0o600); err != nil {
		require.NoError(t, err, "write compose file")
	}

	file, err := Load(composePath)
	require.NoError(t, err, "load compose")
	require.Len(t, file.Services, 1, "unexpected services count")

	service := file.Services[0]
	require.Equal(t, []string{"wmb-infra"}, service.Networks, "service network alias must resolve to name")
	require.Len(t, service.InitJobs, 2, "unexpected init jobs count")
	assert.Nil(t, service.InitJobs[0].Networks, "job without explicit networks must keep nil networks")
	assert.Equal(
		t,
		[]string{"wmb-infra"},
		service.InitJobs[1].Networks,
		"job network alias must resolve to name",
	)
}

func TestLoadParsesServiceDeployReplicasAndSyncPolicy(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yaml")
	composePayload := []byte(`
services:
  api:
    image: ghcr.io/acme/api:1.0.0
    deploy:
      replicas: 3
      labels:
        org.swarm-deploy.service.sync.policy.selfHeal: "false"
`)
	require.NoError(t, os.WriteFile(composePath, composePayload, 0o600), "write compose file")

	file, err := Load(composePath)
	require.NoError(t, err, "load compose")
	require.Len(t, file.Services, 1, "unexpected services count")

	service := file.Services[0]
	require.NotNil(t, service.Replicas, "replicas must be parsed")
	assert.Equal(t, uint64(3), *service.Replicas, "unexpected desired replicas")

	require.NotNil(t, service.SyncPolicy.SelfHeal, "self-heal policy must be parsed")
	assert.False(t, *service.SyncPolicy.SelfHeal, "unexpected self-heal value")
}

func TestLoadFailsOnInvalidSyncPolicyLabelValue(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yaml")
	composePayload := []byte(`
services:
  api:
    image: ghcr.io/acme/api:1.0.0
    deploy:
      labels:
        org.swarm-deploy.service.sync.policy.selfHeal: maybe
`)
	require.NoError(t, os.WriteFile(composePath, composePayload, 0o600), "write compose file")

	_, err := Load(composePath)
	require.Error(t, err, "expected invalid self-heal label error")
	assert.ErrorContains(t, err, "org.swarm-deploy.service.sync.policy.selfHeal", "unexpected error")
}
