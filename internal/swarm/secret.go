package swarm

import (
	"context"
	"fmt"

	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

const secretFileMode = 0o444

type SecretManager struct {
	dockerClient *client.Client
}

func newSecretManager(dockerClient *client.Client) *SecretManager {
	return &SecretManager{
		dockerClient: dockerClient,
	}
}

func (r *SecretManager) ResolveReference(
	ctx context.Context,
	source, target string,
) (*dockerswarm.SecretReference, error) {
	secret, _, err := r.dockerClient.SecretInspectWithRaw(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("inspect secret: %w", err)
	}

	ref := &dockerswarm.SecretReference{
		SecretID:   secret.ID,
		SecretName: secret.Spec.Name,
	}

	if target == "" {
		target = fmt.Sprintf("/run/secrets/%s", ref.SecretName)
	}

	ref.File = &dockerswarm.SecretReferenceFileTarget{
		Name: target,
		UID:  "0",
		GID:  "0",
		Mode: secretFileMode,
	}

	return ref, nil
}
