package registry

import (
	"encoding/base64"
	"testing"

	dockerregistry "github.com/docker/docker/api/types/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const privateImage = "wmb-prod.cr.cloud.ru/services/content-discovery-migrations:latest"

func TestAuthManagerResolveImageUsesAuthFromDockerAuthConfigEnv(t *testing.T) {
	rawAuth := base64.StdEncoding.EncodeToString([]byte("robot:secret"))
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"wmb-prod.cr.cloud.ru":{"auth":"`+rawAuth+`"}}}`)
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	manager := NewAuthManager()
	encodedAuth, err := manager.ResolveImage(privateImage)
	require.NoError(t, err, "resolve image auth")
	require.NotEmpty(t, encodedAuth, "registry auth must be set for private registry")

	authConfig, err := dockerregistry.DecodeAuthConfig(encodedAuth)
	require.NoError(t, err, "decode encoded registry auth")
	assert.Equal(t, "robot", authConfig.Username, "unexpected auth username")
	assert.Equal(t, "secret", authConfig.Password, "unexpected auth password")
	assert.Equal(t, "wmb-prod.cr.cloud.ru", authConfig.ServerAddress, "unexpected registry host")
}

func TestAuthManagerResolveImageReturnsEmptyWithoutAuthConfig(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", "")
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	manager := NewAuthManager()
	encodedAuth, err := manager.ResolveImage(privateImage)
	require.NoError(t, err, "resolve image auth")
	assert.Empty(t, encodedAuth, "registry auth must be empty without auth config")
}

func TestAuthManagerResolveImageUsesConfigLoadedInConstructor(t *testing.T) {
	firstRawAuth := base64.StdEncoding.EncodeToString([]byte("robot:secret"))
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"wmb-prod.cr.cloud.ru":{"auth":"`+firstRawAuth+`"}}}`)
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	manager := NewAuthManager()

	secondRawAuth := base64.StdEncoding.EncodeToString([]byte("robot-updated:secret-updated"))
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":{"wmb-prod.cr.cloud.ru":{"auth":"`+secondRawAuth+`"}}}`)

	encodedAuth, err := manager.ResolveImage(privateImage)
	require.NoError(t, err, "resolve image auth")
	require.NotEmpty(t, encodedAuth, "registry auth must be set for private registry")

	authConfig, err := dockerregistry.DecodeAuthConfig(encodedAuth)
	require.NoError(t, err, "decode encoded registry auth")
	assert.Equal(t, "robot", authConfig.Username, "constructor-loaded config should be used")
	assert.Equal(t, "secret", authConfig.Password, "constructor-loaded config should be used")
}

func TestNewAuthManagerReturnsNopOnInvalidConfig(t *testing.T) {
	t.Setenv("DOCKER_AUTH_CONFIG", `{"auths":`)
	t.Setenv("DOCKER_CONFIG", t.TempDir())

	manager := NewAuthManager()
	_, isNop := manager.(*NopAuthManager)
	require.True(t, isNop, "invalid docker config should fallback to NopAuthManager")

	encodedAuth, err := manager.ResolveImage(privateImage)
	require.NoError(t, err, "nop manager should not return error")
	assert.Empty(t, encodedAuth, "nop manager should not resolve auth")
}
