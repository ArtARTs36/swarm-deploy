package notify

import (
	"context"

	"github.com/artarts36/swarm-deploy/internal/event/dispatcher"
	"github.com/artarts36/swarm-deploy/internal/notify"
)

type Subscriber struct {
	notifier notify.Notifier
}

func NewSubscriber(notifier notify.Notifier) *Subscriber {
	return &Subscriber{
		notifier: notifier,
	}
}

func (s *Subscriber) Handle(ctx context.Context, event dispatcher.Event) error {
	return s.notifier.Notify(ctx, notify.Message{
		Payload: event,
	})
}
