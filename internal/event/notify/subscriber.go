package notify

import (
	"context"

	"github.com/artarts36/swarm-deploy/internal/event/events"
	"github.com/artarts36/swarm-deploy/internal/event/notifiers"
)

type Subscriber struct {
	notifier notifiers.Notifier
}

func NewSubscriber(notifier notifiers.Notifier) *Subscriber {
	return &Subscriber{
		notifier: notifier,
	}
}

func (s *Subscriber) Handle(ctx context.Context, event events.Event) error {
	return s.notifier.Notify(ctx, notifiers.Message{
		Payload: event,
	})
}
