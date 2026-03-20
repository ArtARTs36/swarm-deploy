package event

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/artarts36/swarm-deploy/internal/compose"
	"github.com/artarts36/swarm-deploy/internal/notify"
	"github.com/stretchr/testify/assert"
)

func TestDispatcherDispatchSuccessfulDeployDefersNotifierCall(t *testing.T) {
	notifier := &blockingNotifier{
		started: make(chan notify.Event, 1),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	dispatcher := NewDispatcher(notifier)
	t.Cleanup(func() {
		_ = dispatcher.Shutdown(context.Background())
	})

	finished := make(chan struct{})
	go func() {
		dispatcher.DispatchSuccessfulDeploy(SuccessfulDeployEvent{
			StackName: "prod",
			Commit:    "abc123",
			Services: []compose.Service{
				{
					Name:  "api",
					Image: "ghcr.io/acme/api:1.2.3",
				},
			},
		})
		close(finished)
	}()

	select {
	case <-notifier.started:
	case <-time.After(time.Second):
		t.Fatal("notifier was not called")
	}

	select {
	case <-finished:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("dispatch blocked on notifier call")
	}

	close(notifier.release)

	select {
	case <-notifier.done:
	case <-time.After(time.Second):
		t.Fatal("notifier did not finish after release")
	}
}

func TestDispatcherDispatchFailedDeployDeliversEvent(t *testing.T) {
	notifier := &captureNotifier{
		events: make(chan notify.Event, 1),
	}
	dispatcher := NewDispatcher(notifier)
	t.Cleanup(func() {
		_ = dispatcher.Shutdown(context.Background())
	})

	now := time.Date(2025, time.February, 2, 3, 4, 5, 0, time.UTC)
	dispatcher.now = func() time.Time {
		return now
	}

	dispatcher.DispatchFailedDeploy(FailedDeployEvent{
		StackName: "prod",
		Commit:    "abc123",
		Services: []compose.Service{
			{
				Name: "api",
			},
		},
		Error: errors.New("deploy failed"),
	})

	select {
	case got := <-notifier.events:
		assert.Equal(t, "failed", got.Status)
		assert.Equal(t, "prod", got.StackName)
		assert.Equal(t, "api", got.Service)
		assert.Equal(t, "unknown", got.Image.FullName)
		assert.Equal(t, "unknown", got.Image.Version)
		assert.Equal(t, "abc123", got.Commit)
		assert.Equal(t, "deploy failed", got.Error)
		assert.True(t, now.Equal(got.Timestamp), "unexpected timestamp")
	case <-time.After(time.Second):
		t.Fatal("event was not delivered to notifier")
	}
}

func TestDispatcherLimitsConcurrentNotifierCalls(t *testing.T) {
	const (
		workersCount = 1
		eventsCount  = workersCount + 6
	)

	notifier := &concurrencyNotifier{
		release: make(chan struct{}, eventsCount),
		done:    make(chan struct{}, eventsCount),
	}
	dispatcher := NewDispatcher(notifier)
	t.Cleanup(func() {
		_ = dispatcher.Shutdown(context.Background())
	})

	services := make([]compose.Service, 0, eventsCount)
	for range eventsCount {
		services = append(services, compose.Service{
			Name:  "service",
			Image: "ghcr.io/acme/api:1.2.3",
		})
	}

	finished := make(chan struct{})
	go func() {
		dispatcher.DispatchSuccessfulDeploy(SuccessfulDeployEvent{
			StackName: "prod",
			Commit:    "abc123",
			Services:  services,
		})
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("dispatch blocked unexpectedly")
	}

	for range eventsCount {
		notifier.release <- struct{}{}
	}

	for range eventsCount {
		select {
		case <-notifier.done:
		case <-time.After(time.Second):
			t.Fatal("notifier did not process all events")
		}
	}

	assert.LessOrEqual(t, notifier.maxObserved(), workersCount, "too many concurrent notifier calls")
}

func TestDispatcherShutdownWaitsForActiveNotifierCall(t *testing.T) {
	notifier := &blockingNotifier{
		started: make(chan notify.Event, 1),
		release: make(chan struct{}),
		done:    make(chan struct{}),
	}
	dispatcher := NewDispatcher(notifier)

	dispatcher.DispatchSuccessfulDeploy(SuccessfulDeployEvent{
		StackName: "prod",
		Commit:    "abc123",
		Services: []compose.Service{
			{
				Name:  "api",
				Image: "ghcr.io/acme/api:1.2.3",
			},
		},
	})

	select {
	case <-notifier.started:
	case <-time.After(time.Second):
		t.Fatal("notifier was not called")
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := dispatcher.Shutdown(timeoutCtx)
	cancel()
	assert.Error(t, err, "shutdown should fail with timeout while notifier is blocked")

	close(notifier.release)

	select {
	case <-notifier.done:
	case <-time.After(time.Second):
		t.Fatal("notifier did not finish after release")
	}

	err = dispatcher.Shutdown(context.Background())
	assert.Error(t, err, "second shutdown should fail")
}

func TestDispatcherDispatchAfterShutdownReturnsWithoutNotify(t *testing.T) {
	notifier := &captureNotifier{
		events: make(chan notify.Event, 1),
	}
	dispatcher := NewDispatcher(notifier)

	err := dispatcher.Shutdown(context.Background())
	assert.NoError(t, err, "shutdown should succeed")

	finished := make(chan struct{})
	go func() {
		dispatcher.DispatchSuccessfulDeploy(SuccessfulDeployEvent{
			StackName: "prod",
			Commit:    "abc123",
			Services: []compose.Service{
				{
					Name:  "api",
					Image: "ghcr.io/acme/api:1.2.3",
				},
			},
		})
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("dispatch blocked after shutdown")
	}

	select {
	case <-notifier.events:
		t.Fatal("unexpected notify call after shutdown")
	case <-time.After(50 * time.Millisecond):
	}
}

type blockingNotifier struct {
	started chan notify.Event
	release chan struct{}
	done    chan struct{}
}

func (n *blockingNotifier) Notify(_ context.Context, event notify.Event) error {
	n.started <- event
	<-n.release
	close(n.done)
	return nil
}

type captureNotifier struct {
	events chan notify.Event
}

func (n *captureNotifier) Notify(_ context.Context, event notify.Event) error {
	n.events <- event
	return nil
}

var _ notifier = (*blockingNotifier)(nil)
var _ notifier = (*captureNotifier)(nil)

type concurrencyNotifier struct {
	mu      sync.Mutex
	active  int
	maxSeen int
	release chan struct{}
	done    chan struct{}
}

func (n *concurrencyNotifier) Notify(_ context.Context, _ notify.Event) error {
	n.mu.Lock()
	n.active++
	if n.active > n.maxSeen {
		n.maxSeen = n.active
	}
	n.mu.Unlock()

	<-n.release

	n.mu.Lock()
	n.active--
	n.mu.Unlock()

	n.done <- struct{}{}
	return nil
}

func (n *concurrencyNotifier) maxObserved() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.maxSeen
}

var _ notifier = (*concurrencyNotifier)(nil)
