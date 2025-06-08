package log

import (
	"log/slog"
	"sync"

	"golang.org/x/net/context"
)

type Logger struct {
	logger *slog.Logger
}

func With(args ...any) *Logger {
	return &Logger{
		logger: slog.With(args...),
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
	}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	var newLogger *Logger = l.With()
	if v, ok := ctx.Value(logMapCtxKey).(*sync.Map); ok {
		v.Range(func(key, value any) bool {
			if key, ok := key.(string); ok {
				newLogger = newLogger.With(key, ctx.Value(key))
			}
			return true
		})
	}
	for _, key := range keys {
		if ctx.Value(key) != nil {
			newLogger = newLogger.With(key, ctx.Value(key))
		}
	}
	return newLogger
}
