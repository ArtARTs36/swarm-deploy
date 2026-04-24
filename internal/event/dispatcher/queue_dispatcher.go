package dispatcher

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/swarm-deploy/swarm-deploy/internal/event/events"
)

const (
	defaultEventsQueueLen         = 128
	defaultSubscribeHandleTimeout = 5 * time.Minute
)

type QueueDispatcher struct {
	subscribers map[events.Type][]Subscriber

	now func() time.Time

	queue     chan events.Event
	fastQueue *queue
	slowQueue *queue

	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

const queuesSize = 3

func NewQueueDispatcher() *QueueDispatcher {
	d := &QueueDispatcher{
		now:         time.Now,
		queue:       make(chan events.Event, defaultEventsQueueLen),
		subscribers: map[events.Type][]Subscriber{},
		fastQueue:   newQueue(),
		slowQueue:   newQueue(),
	}

	d.wg.Add(queuesSize)
	go d.runQueueWorker()

	return d
}

func (d *QueueDispatcher) Dispatch(ctx context.Context, event events.Event) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		slog.InfoContext(ctx, "[event] event not dispatched, channel closed", slog.Any("event", event))

		return
	}

	slog.InfoContext(ctx, "[event] dispatching event", slog.Any("event", event),
		slog.String("event.type", event.Type().String()),
	)

	d.queue <- event
}

// Subscribe registers a subscriber for event type.
func (d *QueueDispatcher) Subscribe(eventType events.Type, subscriber Subscriber) {
	d.mu.Lock()
	d.subscribers[eventType] = append(d.subscribers[eventType], subscriber)
	d.mu.Unlock()
}

func (d *QueueDispatcher) runQueueWorker() {
	defer d.wg.Done()

	for event := range d.queue {
		d.mu.RLock()
		subscribers := append([]Subscriber{}, d.subscribers[event.Type()]...)
		d.mu.RUnlock()

		for _, subscriber := range subscribers {
			targetQueue := d.fastQueue

			if subscriber.Slow() {
				targetQueue = d.slowQueue
			}

			targetQueue.Dispatch(&queueTask{
				Event:      event,
				Subscriber: subscriber,
			})
		}
	}
}

func (d *QueueDispatcher) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return errors.New("dispatcher already shut down")
	}
	d.closed = true
	close(d.queue)

	d.slowQueue.Close()
	d.fastQueue.Close()

	d.mu.Unlock()

	waitDone := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		return nil
	case <-ctx.Done():
		return errors.Join(errors.New("shutdown dispatcher"), ctx.Err())
	}
}
