package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutorExecuteUnknownTool(t *testing.T) {
	executor := NewExecutor(&fakeHistoryStore{}, &fakeNodeStore{}, &fakeSyncControl{})

	_, err := executor.Execute(context.Background(), "unknown_tool", nil)
	require.Error(t, err, "expected unknown tool error")
	assert.Contains(t, err.Error(), "unknown tool", "unexpected error")
}
