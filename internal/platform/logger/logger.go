package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger is a thin wrapper around log/slog used across the application.
type Logger struct {
	*slog.Logger
}

// New creates a structured logger using the configured level. Production uses
// JSON output; development uses human-readable text output.
func New(level, environment string) *Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return &Logger{Logger: slog.New(handler)}
}

// NewWithWriter is used by tests.
func NewWithWriter(w io.Writer, level string) *Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return &Logger{Logger: slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl}))}
}

type contextKey struct{}

// WithContext stores the logger on the context.
func WithContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the request logger if present, else a default logger.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok && l != nil {
		return l
	}
	return New("info", "development")
}
