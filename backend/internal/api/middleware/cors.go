package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS returns the CORS middleware configured from the allowed-origins list.
// In production, this is set to https://app.aurganize.work etc.;
// in development, http://localhost:3000 and http://localhost:5173.
//
// Wildcards ("*") are NOT supported because we use credentialed requests
// (HttpOnly refresh-token cookie on web), which CORS forbids combining with
// wildcard origins.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}