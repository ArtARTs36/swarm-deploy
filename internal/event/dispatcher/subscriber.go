package dispatcher

import (
	"context"

	"github.com/artarts36/swarm-deploy/internal/event/events"
)

type Subscriber interface {
	Handle(ctx context.Context, event events.Event) error
}
