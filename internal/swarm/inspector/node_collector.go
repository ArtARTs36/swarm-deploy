package inspector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"
)

const defaultNodeCollectorReconnectDelay = 5 * time.Second

// NodeCollector collects and persists swarm nodes snapshot.
type NodeCollector struct {
	inspector nodeInspector
	store     nodeStoreWriter

	reconnectDelay time.Duration
}

// nodeInspector inspects nodes and watches docker node events.
type nodeInspector interface {
	// InspectNodes returns current swarm nodes snapshot.
	InspectNodes(ctx context.Context) ([]NodeInfo, error)
	// WatchNodeEvents subscribes to Docker node events stream.
	WatchNodeEvents(ctx context.Context) (<-chan dockerevents.Message, <-chan error, error)
}

// nodeStoreWriter saves nodes snapshot.
type nodeStoreWriter interface {
	// Replace replaces current nodes snapshot.
	Replace(nodes []NodeInfo) error
}

// NewNodeCollector creates node collector.
func NewNodeCollector(inspector nodeInspector, store nodeStoreWriter) *NodeCollector {
	return &NodeCollector{
		inspector:      inspector,
		store:          store,
		reconnectDelay: defaultNodeCollectorReconnectDelay,
	}
}

// Run performs initial refresh and subscribes to docker node events.
func (c *NodeCollector) Run(ctx context.Context) error {
	if err := c.refresh(ctx); err != nil {
		slog.WarnContext(ctx, "[nodes] initial refresh failed", slog.Any("err", err))
	}

	for {
		err := c.watchOnce(ctx)
		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		slog.WarnContext(ctx, "[nodes] watch stream failed", slog.Any("err", err))

		timer := time.NewTimer(c.reconnectDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (c *NodeCollector) refresh(ctx context.Context) error {
	nodes, err := c.inspector.InspectNodes(ctx)
	if err != nil {
		return fmt.Errorf("inspect nodes: %w", err)
	}
	if err = c.store.Replace(nodes); err != nil {
		return fmt.Errorf("save nodes snapshot: %w", err)
	}

	slog.InfoContext(ctx, "[nodes] snapshot refreshed", slog.Int("count", len(nodes)))
	return nil
}

func (c *NodeCollector) watchOnce(ctx context.Context) error {
	eventsCh, errorsCh, err := c.inspector.WatchNodeEvents(ctx)
	if err != nil {
		return fmt.Errorf("subscribe docker node events: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-eventsCh:
			if !ok {
				return errors.New("docker node events channel closed")
			}

			slog.DebugContext(ctx, "[nodes] docker node event received",
				slog.String("action", string(event.Action)),
				slog.String("node_id", event.Actor.ID),
			)

			if refreshErr := c.refresh(ctx); refreshErr != nil {
				slog.WarnContext(ctx, "[nodes] refresh after event failed", slog.Any("err", refreshErr))
			}
		case watchErr, ok := <-errorsCh:
			if !ok {
				return errors.New("docker node events errors channel closed")
			}
			if watchErr == nil {
				continue
			}
			return fmt.Errorf("watch docker node events: %w", watchErr)
		}
	}
}
