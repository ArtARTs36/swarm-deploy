package swarm

import (
	"context"
	"errors"
	"fmt"

	cerrdefs "github.com/containerd/errdefs"
	dockerswarm "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// ErrServiceNotFound means that service does not exist in swarm.
var ErrServiceNotFound = errors.New("service not found")

// ServiceManager manages stack service replicas.
type ServiceManager struct {
	dockerClient *client.Client
}

// NewServiceManager creates service manager with provided docker API client.
func NewServiceManager(dockerClient *client.Client) *ServiceManager {
	return &ServiceManager{
		dockerClient: dockerClient,
	}
}

// GetReplicas returns desired replicas count for a stack service.
func (m *ServiceManager) GetReplicas(
	ctx context.Context,
	stackName,
	serviceName string,
) (uint64, error) {
	service, fullServiceName, err := m.inspect(ctx, stackName, serviceName)
	if err != nil {
		return 0, err
	}
	if service.Spec.Mode.Replicated == nil {
		return 0, fmt.Errorf("service %s is not replicated mode", fullServiceName)
	}
	if service.Spec.Mode.Replicated.Replicas == nil {
		return 0, nil
	}

	return *service.Spec.Mode.Replicated.Replicas, nil
}

// Scale sets desired replicas count for a stack service.
func (m *ServiceManager) Scale(
	ctx context.Context,
	stackName,
	serviceName string,
	replicas uint64,
) error {
	service, fullServiceName, err := m.inspect(ctx, stackName, serviceName)
	if err != nil {
		return err
	}
	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return fmt.Errorf("service %s is not replicated mode", fullServiceName)
	}

	spec := service.Spec
	spec.Mode.Replicated.Replicas = &replicas

	_, err = m.dockerClient.ServiceUpdate(ctx, service.ID, service.Version, spec, dockerswarm.ServiceUpdateOptions{})
	if err != nil {
		return fmt.Errorf("update service %s replicas to %d: %w", fullServiceName, replicas, err)
	}

	return nil
}

func (m *ServiceManager) inspect(
	ctx context.Context,
	stackName,
	serviceName string,
) (dockerswarm.Service, string, error) {
	fullServiceName := fmt.Sprintf("%s_%s", stackName, serviceName)
	service, _, err := m.dockerClient.ServiceInspectWithRaw(ctx, fullServiceName, dockerswarm.ServiceInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return dockerswarm.Service{}, fullServiceName, ErrServiceNotFound
		}

		return dockerswarm.Service{}, fullServiceName, fmt.Errorf("inspect service %s: %w", fullServiceName, err)
	}

	return service, fullServiceName, nil
}
