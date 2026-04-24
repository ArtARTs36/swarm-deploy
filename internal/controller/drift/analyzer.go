package drift

import (
	"context"
	"errors"

	"github.com/swarm-deploy/swarm-deploy/internal/compose"
	"github.com/swarm-deploy/swarm-deploy/internal/swarm"
)

const maxUint = ^uint(0)

// ServiceReader reads live service state from runtime.
type ServiceReader interface {
	// GetStatus returns service status by stack/service reference.
	GetStatus(ctx context.Context, serviceRef swarm.ServiceReference) (swarm.ServiceStatus, error)
}

// Analyzer compares desired compose service state with cluster runtime state.
type Analyzer struct {
	services ServiceReader
}

// NewAnalyzer creates drift analyzer.
func NewAnalyzer(services ServiceReader) *Analyzer {
	return &Analyzer{
		services: services,
	}
}

// Analyze detects drift for one service in a stack.
func (a *Analyzer) Analyze(ctx context.Context, stackName string, service compose.Service) (Drift, error) {
	status, err := a.services.GetStatus(ctx, swarm.NewServiceReference(stackName, service.Name))
	if err != nil {
		if errors.Is(err, swarm.ErrServiceNotFound) {
			return Drift{
				OutOfSync:     true,
				ServiceMissed: true,
			}, nil
		}

		return Drift{}, err
	}

	result := Drift{}

	if service.Replicas != nil {
		result.Replicas = Replicas{
			Desired: toUint(*service.Replicas),
			Live:    toUint(status.Spec.Replicas),
		}
		result.Replicas.OutOfSync = result.Replicas.Desired != result.Replicas.Live
	}

	result.OutOfSync = result.ServiceMissed || result.Replicas.OutOfSync

	return result, nil
}

func toUint(value uint64) uint {
	if value > uint64(maxUint) {
		return maxUint
	}

	return uint(value)
}
