package log

import (
	"log/slog"
	"sync"

	"golang.org/x/net/context"
)

type handler struct {
	slog.Handler
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	if v, ok := ctx.Value(logMapCtxKey).(*sync.Map); ok {
		v.Range(func(key, value any) bool {
			if key, ok := key.(string); ok {
				r.AddAttrs(slog.Any(key, value))
			}
			return true
		})
	}
	for _, key := range keys {
		if ctx.Value(key) != nil {
			r.AddAttrs(slog.Any(key, ctx.Value(key)))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}
