package dispatcher

import (
	"context"

	"github.com/artarts36/swarm-deploy/internal/event/events"
)

type Subscriber interface {
	// Name return the subscriber name. Useful for logging purposes.
	Name() string
	Handle(ctx context.Context, event events.Event) error
}
