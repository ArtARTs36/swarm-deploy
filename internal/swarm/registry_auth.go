package swarm

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	dockerregistry "github.com/docker/docker/api/types/registry"
	dockerswarm "github.com/docker/docker/api/types/swarm"
)

const (
	dockerAuthConfigFileName = "config.json"
	usernamePasswordParts    = 2
)

type dockerAuthConfigFile struct {
	Auths map[string]dockerregistry.AuthConfig `json:"auths"`
}

func (d *Deployer) buildInitServiceCreateOptions(image string) (dockerswarm.ServiceCreateOptions, error) {
	encodedRegistryAuth, err := resolveEncodedRegistryAuth(image)
	if err != nil {
		return dockerswarm.ServiceCreateOptions{}, err
	}

	if encodedRegistryAuth == "" {
		return dockerswarm.ServiceCreateOptions{}, nil
	}

	return dockerswarm.ServiceCreateOptions{
		EncodedRegistryAuth: encodedRegistryAuth,
		QueryRegistry:       true,
	}, nil
}

func resolveEncodedRegistryAuth(image string) (string, error) {
	registryHost, err := parseRegistryHost(image)
	if err != nil {
		return "", err
	}

	authConfigFile, found, err := loadDockerAuthConfigFile()
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}

	authConfig, found := findRegistryAuthConfig(authConfigFile.Auths, registryHost)
	if !found {
		return "", nil
	}

	authConfig = decodeAuthField(authConfig)
	if authConfig.ServerAddress == "" {
		authConfig.ServerAddress = registryHost
	}

	encodedRegistryAuth, err := dockerregistry.EncodeAuthConfig(authConfig)
	if err != nil {
		return "", fmt.Errorf("encode registry auth: %w", err)
	}
	return encodedRegistryAuth, nil
}

func parseRegistryHost(image string) (string, error) {
	if strings.TrimSpace(image) == "" {
		return "", nil
	}

	named, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", fmt.Errorf("parse image reference %q: %w", image, err)
	}

	return reference.Domain(named), nil
}

func loadDockerAuthConfigFile() (dockerAuthConfigFile, bool, error) {
	dockerAuthConfigRaw := strings.TrimSpace(os.Getenv("DOCKER_AUTH_CONFIG"))
	if dockerAuthConfigRaw == "" {
		configPath, ok, err := resolveDockerConfigPath()
		if err != nil {
			return dockerAuthConfigFile{}, false, err
		}
		if !ok {
			return dockerAuthConfigFile{}, false, nil
		}

		configBytes, readErr := os.ReadFile(configPath)
		if errors.Is(readErr, os.ErrNotExist) {
			return dockerAuthConfigFile{}, false, nil
		}
		if readErr != nil {
			return dockerAuthConfigFile{}, false, fmt.Errorf("read docker auth config %s: %w", configPath, readErr)
		}

		dockerAuthConfigRaw = string(configBytes)
	}

	authConfig := dockerAuthConfigFile{}
	if err := json.Unmarshal([]byte(dockerAuthConfigRaw), &authConfig); err != nil {
		return dockerAuthConfigFile{}, false, fmt.Errorf("decode docker auth config: %w", err)
	}

	if len(authConfig.Auths) == 0 {
		return dockerAuthConfigFile{}, false, nil
	}

	return authConfig, true, nil
}

func resolveDockerConfigPath() (string, bool, error) {
	if dockerConfigDir := strings.TrimSpace(os.Getenv("DOCKER_CONFIG")); dockerConfigDir != "" {
		return filepath.Join(dockerConfigDir, dockerAuthConfigFileName), true, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve home directory for docker config: %w", err)
	}
	if homeDir == "" {
		return "", false, nil
	}

	return filepath.Join(homeDir, ".docker", dockerAuthConfigFileName), true, nil
}

func findRegistryAuthConfig(
	authConfigs map[string]dockerregistry.AuthConfig,
	registryHost string,
) (dockerregistry.AuthConfig, bool) {
	normalizedRegistryHost := normalizeRegistryHost(registryHost)
	for serverAddress, authConfig := range authConfigs {
		if normalizeRegistryHost(serverAddress) != normalizedRegistryHost {
			continue
		}
		if authConfig.ServerAddress == "" {
			authConfig.ServerAddress = registryHost
		}
		return authConfig, true
	}
	return dockerregistry.AuthConfig{}, false
}

func normalizeRegistryHost(v string) string {
	normalized := strings.TrimSpace(strings.ToLower(v))
	normalized = strings.TrimPrefix(normalized, "https://")
	normalized = strings.TrimPrefix(normalized, "http://")
	normalized = strings.TrimSuffix(normalized, "/")
	if slashIdx := strings.Index(normalized, "/"); slashIdx >= 0 {
		normalized = normalized[:slashIdx]
	}

	switch normalized {
	case "index.docker.io", "registry-1.docker.io":
		return "docker.io"
	default:
		return normalized
	}
}

func decodeAuthField(authConfig dockerregistry.AuthConfig) dockerregistry.AuthConfig {
	if authConfig.Auth == "" || (authConfig.Username != "" && authConfig.Password != "") {
		return authConfig
	}

	decodedRaw, err := base64.StdEncoding.DecodeString(authConfig.Auth)
	if err != nil {
		decodedRaw, err = base64.RawStdEncoding.DecodeString(authConfig.Auth)
		if err != nil {
			return authConfig
		}
	}

	usernameAndPassword := strings.SplitN(string(decodedRaw), ":", usernamePasswordParts)
	if len(usernameAndPassword) != usernamePasswordParts {
		return authConfig
	}

	authConfig.Username = usernameAndPassword[0]
	authConfig.Password = usernameAndPassword[1]
	return authConfig
}
