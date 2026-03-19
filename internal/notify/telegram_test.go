package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelegramNotifierSendsThreadAndRenderedTemplate(t *testing.T) {
	var received map[string]any

	notifier, err := NewTelegramNotifier(
		"ops",
		"TOKEN",
		"-100123",
		TelegramOptions{
			ChatThreadID: 42,
			Message:      "stack={{.stack_name}} image={{.image.full_name}}:{{.image.version}} success={{.success}}",
			APIBaseURL:   "https://telegram.invalid",
		},
	)
	require.NoError(t, err, "build notifier")
	notifier.client = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodPost, r.Method, "unexpected method")
			require.Equal(t, "/botTOKEN/sendMessage", r.URL.Path, "unexpected path")

			defer r.Body.Close()
			decodeErr := json.NewDecoder(r.Body).Decode(&received)
			require.NoError(t, decodeErr, "decode request body")

			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	err = notifier.Notify(context.Background(), Event{
		Status:    "success",
		StackName: "app",
		Service:   "api",
		Image: Image{
			FullName: "ghcr.io/acme/api",
			Version:  "1.2.3",
		},
		Timestamp: time.Now(),
	})
	require.NoError(t, err, "notify")

	assert.Equal(t, "-100123", received["chat_id"], "unexpected chat_id")
	threadIDRaw, ok := received["message_thread_id"].(float64)
	require.True(t, ok, "message_thread_id has unexpected type: %#v", received["message_thread_id"])
	assert.Equal(t, int64(42), int64(threadIDRaw), "unexpected message_thread_id")
	assert.Equal(t, "stack=app image=ghcr.io/acme/api:1.2.3 success=true", received["text"], "unexpected text")
}

func TestTelegramNotifierInvalidTemplate(t *testing.T) {
	_, err := NewTelegramNotifier(
		"ops",
		"TOKEN",
		"-100123",
		TelegramOptions{
			Message: "{{ if }}",
		},
	)
	require.Error(t, err, "expected error")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
