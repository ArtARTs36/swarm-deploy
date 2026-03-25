package logx

import (
	"context"
	"log/slog"

	"github.com/artarts36/swarm-deploy/internal/event/events"
	"github.com/cappuccinotm/slogx"
)

type eventTypeKey struct{}

func ContextWithEventType(ctx context.Context, eventType events.Type) context.Context {
	return context.WithValue(ctx, eventTypeKey{}, eventType)
}

func EventTypeFromContext(ctx context.Context) (events.Type, bool) {
	typ, ok := ctx.Value(eventTypeKey{}).(events.Type)
	if ok {
		return typ, true
	}
	return "", false
}

func EventType() slogx.Middleware {
	return func(next slogx.HandleFunc) slogx.HandleFunc {
		return func(ctx context.Context, rec slog.Record) error {
			if typ, ok := EventTypeFromContext(ctx); ok {
				rec.AddAttrs(slog.String("event.type", string(typ)))
			}
			return next(ctx, rec)
		}
	}
}
