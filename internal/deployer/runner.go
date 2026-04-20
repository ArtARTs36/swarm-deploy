package deployer

import "context"

type Runner interface {
	Run(ctx context.Context, args ...string) (string, error)
}
