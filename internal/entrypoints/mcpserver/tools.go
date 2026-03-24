package mcpserver

import mcpTools "github.com/artarts36/swarm-deploy/internal/entrypoints/mcpserver/tools"

// Tools provides direct-call MCP tools without running external server.
type Tools = mcpTools.Executor

// NewTools creates a tool executor.
func NewTools(
	historyStore mcpTools.HistoryReader,
	nodesStore mcpTools.NodesReader,
	control mcpTools.SyncTrigger,
) *Tools {
	return mcpTools.NewExecutor(historyStore, nodesStore, control)
}
