package tools

import (
	"github.com/artarts36/swarm-deploy/internal/controller"
	"github.com/artarts36/swarm-deploy/internal/event/history"
	"github.com/artarts36/swarm-deploy/internal/swarm"
)

// HistoryReader reads current event history snapshot.
type HistoryReader interface {
	// List returns current event history snapshot.
	List() []history.Entry
}

// SyncTrigger triggers synchronization run.
type SyncTrigger interface {
	// Trigger enqueues synchronization by reason.
	Trigger(reason controller.TriggerReason) bool
}

// NodesReader reads current Swarm nodes snapshot.
type NodesReader interface {
	// List returns current nodes snapshot.
	List() []swarm.NodeInfo
}
