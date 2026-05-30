package main

import (
	"log/slog"
	"net/http"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/middleware"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// buildRouter wires up all routes and middleware. We re-build this in
// 05_middleware.md and 06_auth_endpoints.md as we add layers.
func buildRouter(cfg *config.Config, logger *slog.Logger, pool *pgxpool.Pool) http.Handler {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	jwtSvc := auth.NewJWTservice(cfg.JWTSecret, cfg.JWTAccessTokenTTLWeb, cfg.JWTAccessTokneTTLMobile)

	// gin.New() instead of gin.Default() — Default() adds gin's stdout
	// logger and recovery middleware in a format that doesn't match slog.
	// We add our own (file 05).
	r := gin.New()

	// === Global Middleware pipeline (part of every request pipeline) ===
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// ** Replaced the gin recovery middleware with our custom version **
	// // Built-in panic recovery so a handler bug doesn't crash the server.
	// // Replaced with a custom slog-integrated version in file 05.
	// r.Use(gin.Recovery())

	// === Health Check ===
	// Public, unauthenticated. Used by uptime monitors and Fly's
	// load balancer. Returns 200 if the process is up and the DB is
	// reachable; 503 otherwise.
	r.GET("/api/v1/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"db":     "unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})

	})

	// === Public auth endpoints (will be filled in 06_auth_endpoints.md) ===
	// public := r.Group("/api/v1/auth")
	// {
	//     public.POST("/signup", ...)
	//     public.POST("/login", ...)
	//     public.POST("/refresh", ...)
	// }

	// === Protected endpoints ===
	protected := r.Group("/api/v1")
	protected.Use(middleware.Auth(jwtSvc))
	protected.Use(middleware.Tenancy(pool))

	// A minimal "who am I?" endpoint to verify the middleware chain works.
	// Replaced/extended in 06.
	protected.GET("/me", func(ctx *gin.Context) {
		a := middleware.MustAuth(ctx)
		ctx.JSON(http.StatusOK, gin.H{
			"user_id":   a.UserId,
			"tenant_id": a.TenantId,
			"role":      a.Role,
			"client":    a.Client,
		})
	})

	return r
}
