package tools

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncExecute(t *testing.T) {
	control := &fakeSyncControl{queued: true}
	tool := NewSync(control)

	raw, err := tool.Execute(nil)
	require.NoError(t, err, "execute sync tool")

	var payload struct {
		Queued bool `json:"queued"`
	}

	require.NoError(t, json.Unmarshal([]byte(raw), &payload), "decode response")
	assert.True(t, payload.Queued, "expected queued=true response")
	assert.Equal(t, 1, control.called, "expected single trigger call")
}
