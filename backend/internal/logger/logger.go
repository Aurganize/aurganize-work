package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// contextKey is a private type used as map key for context-stored logger
// fields. Private prevents collisions with other packages' context keys.
type contextKey struct{ name string }

var loggerKey = contextKey{name: "logger"}

// New constructs a slog.Logger with the configured level and format.
// Pass io.Discard as `out` in tests; pass os.Stdout in production.
func New(level string, format string, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}

	opts := slog.HandlerOptions{
		Level: parseLevel(level),
		// ReplaceAttr lets us rename built-in attribute keys.
		// We rename "msg" -> "message" for consistency with most log
		// aggregators (Sentry, Datadog, etc.) that expect "message".
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	}

	var handler slog.Handler
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(out, &opts)
	} else {
		handler = slog.NewJSONHandler(out, &opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info", "information":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a new context carrying the given logger.
// Use this to inject a per-request logger that already has request_id,
// user_id, tenant_id pre-attached.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext extracts the logger from a context. If none is set,
// returns the default logger. Never returns nil.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok && l != nil {
		return l
	}

	return slog.Default()
}
