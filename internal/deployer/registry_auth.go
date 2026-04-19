package deployer

import (
	dockerswarm "github.com/docker/docker/api/types/swarm"
)

func (d *Deployer) buildInitServiceCreateOptions(image string) (dockerswarm.ServiceCreateOptions, error) {
	encodedRegistryAuth, err := d.authManager.ResolveImage(image)
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
