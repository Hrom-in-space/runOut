package logger

import (
	"context"
	"log/slog"
)

type Logger string

const LoggerKey = Logger("Logger")

func ToCtx(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

func FromCtx(ctx context.Context) *slog.Logger {
	v := ctx.Value(LoggerKey)
	if v == nil {
		return slog.Default()
	}

	data, ok := v.(*slog.Logger)
	if !ok {
		return slog.Default()
	}

	return data
}
