package deployer

import "context"

type Runner interface {
	Run(ctx context.Context, command string, args ...string) (string, error)
}
