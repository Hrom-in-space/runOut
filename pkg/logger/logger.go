package logger

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

func New(level string, format string) (*slog.Logger, error) {
	slogLevel, err := ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slogLevel,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// заменяем source со структурой на строку shortPath:line
			if attr.Key == slog.SourceKey {
				if src, ok := attr.Value.Any().(*slog.Source); ok {
					return slog.Attr{
						Key:   "source",
						Value: slog.StringValue(fmt.Sprintf("%s:%d", TrimPath(src.File), src.Line)),
					}
				}
			}

			return attr
		},
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	case "color":
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.TimeOnly,
		})
	default:
		return nil, errors.New("unknown log format")
	}

	return slog.New(handler), nil
}

// TrimPath возвращает dir/package/file из пути файла.
func TrimPath(path string) string {
	const maxPathLen = 3
	var pathLen int
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			pathLen++
			if pathLen == maxPathLen {
				return path[i+1:]
			}
		}
	}

	return path
}

// ParseLevel возвращает slog.Level из строки.
func ParseLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelDebug, errors.New("unknown log level")
	}
}

func Error(err error) slog.Attr {
	return tint.Err(err)
}
