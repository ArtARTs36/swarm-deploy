package swarm

import (
	"context"
	"fmt"
	"sort"
	"time"

	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

const secretFileMode = 0o444

type SecretManager struct {
	dockerClient *client.Client
}

type Secret struct {
	// ID is a unique Docker secret identifier.
	ID string `json:"id"`
	// Name is a Docker secret name.
	Name string `json:"name"`
	// CreatedAt is a secret creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is a secret update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
	// Driver is an external secret driver name when configured.
	Driver string `json:"driver"`
	// Labels contains custom Docker secret labels.
	Labels map[string]string `json:"labels"`
}

func newSecretManager(dockerClient *client.Client) *SecretManager {
	return &SecretManager{
		dockerClient: dockerClient,
	}
}

func (r *SecretManager) List(ctx context.Context) ([]Secret, error) {
	secrets, err := r.dockerClient.SecretList(ctx, dockerswarm.SecretListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list docker secrets: %w", err)
	}

	mapped := make([]Secret, len(secrets))
	for i, secret := range secrets {
		mapped[i] = r.mapSecretInfo(secret)
	}
	r.sortSecretInfos(mapped)

	return mapped, nil
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

func (*SecretManager) mapSecretInfo(secret dockerswarm.Secret) Secret {
	driver := ""
	if secret.Spec.Driver != nil {
		driver = secret.Spec.Driver.Name
	}

	return Secret{
		ID:        secret.ID,
		Name:      secret.Spec.Name,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
		Driver:    driver,
		Labels:    secret.Spec.Labels,
	}
}

func (*SecretManager) sortSecretInfos(secrets []Secret) {
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Name != secrets[j].Name {
			return secrets[i].Name < secrets[j].Name
		}

		return secrets[i].ID < secrets[j].ID
	})
}
