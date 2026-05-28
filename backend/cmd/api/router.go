package main

import (
	"log/slog"
	"net/http"

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

	// gin.New() instead of gin.Default() — Default() adds gin's stdout
	// logger and recovery middleware in a format that doesn't match slog.
	// We add our own (file 05).
	r := gin.New()

	// Built-in panic recovery so a handler bug doesn't crash the server.
	// Replaced with a custom slog-integrated version in file 05.
	r.Use(gin.Recovery())

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
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": "0.1.0",
		})

	})

	return r
}
