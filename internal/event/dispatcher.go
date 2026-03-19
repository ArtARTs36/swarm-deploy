package event

import (
	"context"
	"log/slog"
	"time"

	"github.com/artarts36/swarm-deploy/internal/compose"
	"github.com/artarts36/swarm-deploy/internal/notify"
)

type notifier interface {
	Notify(ctx context.Context, event notify.Event) error
}

type Dispatcher struct {
	notifier notifier
	now      func() time.Time
}

func NewDispatcher(notifier notifier) *Dispatcher {
	return &Dispatcher{
		notifier: notifier,
		now:      time.Now,
	}
}

func (d *Dispatcher) DispatchSuccessfulDeploy(event SuccessfulDeployEvent) {
	for _, service := range event.Services {
		imageName := service.Image
		if imageName == "" {
			imageName = "unknown"
		}

		d.dispatch(notify.Event{
			Status:    "success",
			StackName: event.StackName,
			Service:   service.Name,
			Image: notify.Image{
				FullName: imageName,
				Version:  compose.ImageVersion(imageName),
			},
			Commit:    event.Commit,
			Timestamp: d.now(),
		})
	}
}

func (d *Dispatcher) DispatchFailedDeploy(
	event FailedDeployEvent,
) {
	for _, service := range event.Services {
		d.dispatch(notify.Event{
			Status:    "failed",
			StackName: event.StackName,
			Service:   service.Name,
			Image: notify.Image{
				FullName: "unknown",
				Version:  "unknown",
			},
			Commit:    event.Commit,
			Error:     event.Error.Error(),
			Timestamp: d.now(),
		})
	}
}

func (d *Dispatcher) dispatch(event notify.Event) {
	ctx := context.Background()
	if err := d.notifier.Notify(ctx, event); err != nil {
		slog.ErrorContext(ctx, "[event] failed to notify", slog.Any("err", err))
	}
}
