package middleware

import (
	"errors"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer "

// Auth returns a middleware that demands a valid access token.
// On success, it sets the AuthCtx on both gin and context.
// On failure, it returns 401 and aborts the chain.
//
// Use Auth on the protected route group; do NOT apply globally
// (login/signup/refresh must be reachable without a token).
func Auth(jwtSvc *auth.JWTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" {
			response.RenderError(
				ctx, domain.ErrUnauthenticated("missing Authorization header", nil),
			)
			return
		}
		if !strings.HasPrefix(header, bearerPrefix) {
			response.RenderError(ctx, domain.ErrUnauthenticated("Authorization header must use Bearer scheme", nil))
		}

		tokenStr := strings.TrimPrefix(header, bearerPrefix)
		claims, err := jwtSvc.ParseAccessToken(tokenStr)
		if err != nil {
			// Any parse failure (signature, expiry, format) → opaque 401.
			// The exact reason is logged by ParseAccessToken upstream; we
			// never tell the client which check failed.
			if errors.Is(err, auth.ErrInvalidToken) {
				response.RenderError(ctx, domain.ErrUnauthenticated("invalid or expired token", err))
			}
			response.RenderError(ctx, domain.ErrUnauthenticated("token parse error", err))
			return
		}

		setAuthCtx(ctx, &AuthContext{
			TenantId: claims.TenantId,
			UserId:   claims.UserId,
			Role:     claims.Role,
			Client:   claims.Client.ToString(),
		})

		ctx.Next()
	}
}
