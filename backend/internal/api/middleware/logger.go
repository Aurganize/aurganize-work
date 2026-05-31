package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware that logs each request after it
// completes, attaching status, latency, method, path, and the request ID.
//
// Logging happens *after* the handler runs (in deferred form would be
// equivalent), so the status code recorded is the final one written.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		raw := ctx.Request.URL.RawQuery

		// SetLogger(ctx, logger)
		// Run the rest of the chain.
		ctx.Next()

		latency := time.Since(start)
		status := ctx.Writer.Status()

		// Skip the noisy 200s on /health if they're frequent enough to clutter the log.
		// Comment out the next block if you want every request logged.
		if path == "/api/v1/health" && status == 200 {
			return
		}

		if raw != "" {
			path = path + "?" + raw
		}

		attrs := []slog.Attr{
			slog.String("request_id", GetRequestID(ctx.Request.Context())),
			slog.String("method", ctx.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", ctx.ClientIP()),
		}

		// Attach auth context if present — useful for "who made this request?".
		if a := GetAuthCtx(ctx.Request.Context()); a != nil {
			attrs = append(attrs,
				slog.String("tenant_id", a.TenantId.String()),
				slog.String("user_id", a.UserId.String()),
				slog.String("role", a.Role),
			)
		}

		// Pick level by status.
		switch {
		case status >= 500:
			logger.LogAttrs(ctx.Request.Context(), slog.LevelError, "request", attrs...)
		case status >= 400:
			logger.LogAttrs(ctx.Request.Context(), slog.LevelWarn, "request", attrs...)
		default:
			logger.LogAttrs(ctx.Request.Context(), slog.LevelInfo, "request", attrs...)
		}
	}
}
