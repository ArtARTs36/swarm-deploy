package inspector

import (
	"context"
	"fmt"
	"sort"

	"github.com/artarts36/swarm-deploy/internal/swarm"
	dockerswarm "github.com/docker/docker/api/types/swarm"
)

// InspectSecrets returns current Docker secrets snapshot.
func (i *Inspector) InspectSecrets(ctx context.Context) ([]swarm.Secret, error) {
	secrets, err := i.dockerClient.SecretList(ctx, dockerswarm.SecretListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list docker secrets: %w", err)
	}

	mapped := make([]swarm.Secret, 0, len(secrets))
	for _, secret := range secrets {
		mapped = append(mapped, toSecret(secret))
	}
	sortSecrets(mapped)

	return mapped, nil
}

func toSecret(secret dockerswarm.Secret) swarm.Secret {
	driver := ""
	if secret.Spec.Driver != nil {
		driver = secret.Spec.Driver.Name
	}

	return swarm.Secret{
		ID:        secret.ID,
		Name:      secret.Spec.Name,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
		Driver:    driver,
		Labels:    secret.Spec.Labels,
	}
}

func sortSecrets(secrets []swarm.Secret) {
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Name != secrets[j].Name {
			return secrets[i].Name < secrets[j].Name
		}

		return secrets[i].ID < secrets[j].ID
	})
}
