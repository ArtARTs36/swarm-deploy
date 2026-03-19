package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWithStacksFile(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
  - name: worker
    composeFile: worker/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
sync:
  mode: pull
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")
	require.Len(t, cfg.Spec.Stacks, 2, "expected 2 stacks")
	assert.Equal(t, "app", cfg.Spec.Stacks[0].Name, "unexpected first stack")
	assert.Equal(t, "worker", cfg.Spec.Stacks[1].Name, "unexpected second stack")
}

func TestLoadFailsWithoutStacksFile(t *testing.T) {
	dir := t.TempDir()

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
sync:
  mode: pull
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	_, err := Load(configPath)
	require.Error(t, err, "expected error")
	assert.Contains(t, err.Error(), "stacksFile is required", "unexpected error")
}

func TestWebhookSecretResolveFromFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte(" from-file \n"), 0o600); err != nil {
		require.NoError(t, err, "write secret file")
	}

	spec := WebhookSpec{
		SecretPath: secretPath,
	}

	assert.Equal(t, "from-file", spec.ResolveSecret(), "expected secret from file")
}

func TestLoadResolvesRelativeWebhookSecretPath(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte("from-file"), 0o600); err != nil {
		require.NoError(t, err, "write secret file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
sync:
  mode: webhook
  webhook:
    enabled: true
    secretPath: ./webhook_secret
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")
	assert.Equal(t, secretPath, cfg.Spec.Sync.Webhook.SecretPath, "expected resolved secretPath")
	assert.Equal(t, "from-file", cfg.WebhookSecret(), "expected secret from file")
}

func TestLoadIgnoresDataDirFromConfig(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte("secret"), 0o600); err != nil {
		require.NoError(t, err, "write secret file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
dataDir: /tmp/custom-path-should-be-ignored
git:
  repository: https://example.com/repo.git
sync:
  mode: webhook
  webhook:
    enabled: true
    secretPath: ./webhook_secret
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")

	expectedDataDir := filepath.Join(dir, ".swarm-deploy")
	assert.Equal(t, expectedDataDir, cfg.Spec.DataDir, "expected dataDir")
}

func TestLoadWebAddressUsedForSingleServer(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
stacksFile: ./stacks.yaml
web:
  address: ":18080"
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")
	assert.Equal(t, ":18080", cfg.Spec.Web.Address, "expected web.address")
	assert.Equal(t, defaultWebhookAddress, cfg.Spec.Sync.Webhook.Address, "expected sync.webhook.address")
}

func TestLoadWebAddressDefaults(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")
	assert.Equal(t, defaultWebAddress, cfg.Spec.Web.Address, "expected web.address")
	assert.Equal(t, defaultWebhookAddress, cfg.Spec.Sync.Webhook.Address, "expected sync.webhook.address")
}

func TestLoadResolvesRelativeGitSSHPassphrasePath(t *testing.T) {
	dir := t.TempDir()

	stacksPath := filepath.Join(dir, "stacks.yaml")
	stacksPayload := []byte(`
stacks:
  - name: app
    composeFile: app/docker-compose.yml
`)
	if err := os.WriteFile(stacksPath, stacksPayload, 0o600); err != nil {
		require.NoError(t, err, "write stacks file")
	}

	passphrasePath := filepath.Join(dir, "git_passphrase")
	if err := os.WriteFile(passphrasePath, []byte(" super-secret \n"), 0o600); err != nil {
		require.NoError(t, err, "write passphrase file")
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: git@github.com:your-org/your-stacks-repo.git
  auth:
    type: ssh
    ssh:
      privateKeyPath: /run/secrets/deploy_key
      passphrasePath: ./git_passphrase
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		require.NoError(t, err, "write config file")
	}

	cfg, err := Load(configPath)
	require.NoError(t, err, "load config")
	assert.Equal(t, passphrasePath, cfg.Spec.Git.Auth.SSH.PassphrasePath, "expected passphrasePath")

	passphrase, err := cfg.Spec.Git.Auth.SSH.ResolvePassphrase()
	require.NoError(t, err, "resolve passphrase")
	assert.Equal(t, "super-secret", passphrase, "expected passphrase")
}
