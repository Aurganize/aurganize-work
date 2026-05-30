package middleware

import (
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer"

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
	}
}
