package middleware

import (
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"github.com/gin-gonic/gin"
)

// RequireRole returns a middleware that allows the request through only
// if the authenticated user's role is in the given allowlist.
//
// Usage:
//
//	protected := r.Group("/api/v1")
//	protected.Use(AuthRequired(jwt), Tenancy(pool))
//	protected.POST("/users/invite", RequireRole("admin"), userHandler.Invite)
//
// Multiple roles allowed: RequireRole("admin", "finance").
func RequireRole(allowed ...string) gin.HandlerFunc {
	allowSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowSet[r] = struct{}{}
	}

	return func(ctx *gin.Context) {
		a := GetAuthCtx(ctx.Request.Context())
		if a == nil {
			response.RenderError(ctx, domain.ErrUnauthenticated("not authenticated", nil))
			return
		}

		if _, ok := allowSet[a.Role]; !ok {
			response.RenderError(ctx, domain.ErrForbidden("insufficent role", nil))
			return
		}

		ctx.Next()
	}
}
