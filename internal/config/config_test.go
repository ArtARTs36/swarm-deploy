package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		t.Fatalf("write stacks file: %v", err)
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
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Spec.Stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(cfg.Spec.Stacks))
	}
	if cfg.Spec.Stacks[0].Name != "app" || cfg.Spec.Stacks[1].Name != "worker" {
		t.Fatalf("unexpected stacks order/content: %+v", cfg.Spec.Stacks)
	}
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
		t.Fatalf("write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stacksFile is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhookSecretResolveFromFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte(" from-file \n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	spec := WebhookSpec{
		SecretPath: secretPath,
	}

	if got := spec.ResolveSecret(); got != "from-file" {
		t.Fatalf("expected secret from file, got %q", got)
	}
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
		t.Fatalf("write stacks file: %v", err)
	}

	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
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
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Spec.Sync.Webhook.SecretPath != secretPath {
		t.Fatalf("expected resolved secretPath %q, got %q", secretPath, cfg.Spec.Sync.Webhook.SecretPath)
	}
	if got := cfg.WebhookSecret(); got != "from-file" {
		t.Fatalf("expected secret from file, got %q", got)
	}
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
		t.Fatalf("write stacks file: %v", err)
	}

	secretPath := filepath.Join(dir, "webhook_secret")
	if err := os.WriteFile(secretPath, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
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
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expectedDataDir := filepath.Join(dir, ".swarm-deploy")
	if cfg.Spec.DataDir != expectedDataDir {
		t.Fatalf("expected dataDir %q, got %q", expectedDataDir, cfg.Spec.DataDir)
	}
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
		t.Fatalf("write stacks file: %v", err)
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
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Spec.Web.Address != ":18080" {
		t.Fatalf("expected web.address :18080, got %q", cfg.Spec.Web.Address)
	}
	if cfg.Spec.Sync.Webhook.Address != defaultWebhookAddress {
		t.Fatalf("expected sync.webhook.address %s, got %q", defaultWebhookAddress, cfg.Spec.Sync.Webhook.Address)
	}
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
		t.Fatalf("write stacks file: %v", err)
	}

	configPath := filepath.Join(dir, "swarm-deploy.yaml")
	configPayload := []byte(`
git:
  repository: https://example.com/repo.git
stacksFile: ./stacks.yaml
`)
	if err := os.WriteFile(configPath, configPayload, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Spec.Web.Address != defaultWebAddress {
		t.Fatalf("expected web.address %s, got %q", defaultWebAddress, cfg.Spec.Web.Address)
	}
	if cfg.Spec.Sync.Webhook.Address != defaultWebhookAddress {
		t.Fatalf("expected sync.webhook.address %s, got %q", defaultWebhookAddress, cfg.Spec.Sync.Webhook.Address)
	}
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
		t.Fatalf("write stacks file: %v", err)
	}

	passphrasePath := filepath.Join(dir, "git_passphrase")
	if err := os.WriteFile(passphrasePath, []byte(" super-secret \n"), 0o600); err != nil {
		t.Fatalf("write passphrase file: %v", err)
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
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Spec.Git.Auth.SSH.PassphrasePath != passphrasePath {
		t.Fatalf("expected passphrasePath %q, got %q", passphrasePath, cfg.Spec.Git.Auth.SSH.PassphrasePath)
	}

	passphrase, err := cfg.Spec.Git.Auth.SSH.ResolvePassphrase()
	if err != nil {
		t.Fatalf("resolve passphrase: %v", err)
	}
	if passphrase != "super-secret" {
		t.Fatalf("expected passphrase super-secret, got %q", passphrase)
	}
}
