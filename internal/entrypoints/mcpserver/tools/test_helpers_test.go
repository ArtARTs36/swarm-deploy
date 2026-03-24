package tools

import (
	"github.com/artarts36/swarm-deploy/internal/controller"
	"github.com/artarts36/swarm-deploy/internal/event/history"
	"github.com/artarts36/swarm-deploy/internal/swarm"
)

type fakeHistoryStore struct {
	entries []history.Entry
}

func (f *fakeHistoryStore) List() []history.Entry {
	out := make([]history.Entry, len(f.entries))
	copy(out, f.entries)

	return out
}

type fakeSyncControl struct {
	queued bool
	called int
}

func (f *fakeSyncControl) Trigger(_ controller.TriggerReason) bool {
	f.called++

	return f.queued
}

type fakeNodeStore struct {
	nodes []swarm.NodeInfo
}

func (f *fakeNodeStore) List() []swarm.NodeInfo {
	out := make([]swarm.NodeInfo, len(f.nodes))
	copy(out, f.nodes)

	return out
}
