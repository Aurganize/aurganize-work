package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-ID"

// RequestID assigns each request a stable identifier (UUIDv4) used for
// log correlation. If the client provides X-Request-ID, we honour it
// (useful for distributed tracing). The chosen ID is echoed back as
// X-Request-ID on the response.
//
// Note: gin-contrib/requestid does the same thing but it stores the ID in
// gin's c.Get only, not on the standard context. We need both, so we roll
// our own.
func RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id := ctx.GetHeader(headerRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		SetRequestID(ctx, id)
		ctx.Header(headerRequestID, id)
		ctx.Next()
	}
}
