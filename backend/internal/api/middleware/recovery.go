package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"github.com/gin-gonic/gin"
)

// Recovery returns a gin middleware that catches panics from downstream
// handlers/middleware and turns them into a clean 500 response. The panic
// value and stack trace are logged at error level.
//
// Without this, a panic propagates to gin's default handler which prints
// to stderr in non-JSON format — fine in dev, useless in prod log streams.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Build an error from the recovered value.
				var err error
				switch v := rec.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("panic: %v", v)
				}

				logger.Error("handler panic",
					slog.String("request_id", GetRequestID(ctx)),
					slog.String("path", ctx.Request.RequestURI),
					slog.String("method", ctx.Request.Method),
					slog.Any("err", err),
					slog.String("stack", string(debug.Stack())))

				_ = ctx.AbortWithError(http.StatusInternalServerError, err)
				response.RenderError(ctx, domain.ErrInternal(err))
			}
		}()

		ctx.Next()
	}
}
