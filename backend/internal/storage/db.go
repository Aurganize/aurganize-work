package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool constructs a pgxpool.Pool from a Postgres URL string.
// Configures sensible defaults for a SaaS workload: 25 max conns,
// 5-minute idle timeout, 1-hour max conn lifetime.
//
// Call once in main(); pass the *pgxpool.Pool by dependency injection
// into handlers and services. Close it in main() via defer.
func NewPool(c context.Context, databaseURL string) (*pgxpool.Pool, error) {
	dbConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Pool sizing — start small, scale by observing.
	// 25 conns × N pods should comfortably handle pilot traffic; Neon's
	// Launch tier supports ~100 connections, so 25 leaves headroom.
	dbConfig.MaxConns = 25
	dbConfig.MinConns = 2
	dbConfig.MaxConnIdleTime = 5 * time.Minute
	dbConfig.MaxConnLifetime = 1 * time.Hour
	dbConfig.HealthCheckPeriod = 30 * time.Second

	// AfterRelease runs every time a connection is returned to the pool.
	// We reset app.tenant_id so the connection is clean for whoever
	// borrows it next. Defence in depth — middleware should already
	// have used SET LOCAL inside a transaction (which auto-resets).
	dbConfig.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(), "SELECT set_config('app.tenant_id','' , false)")
		if err != nil {
			// If we can't reset, drop the connection rather than reuse it.
			return false
		}
		return true
	}

	pool, err := pgxpool.NewWithConfig(c, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("creating pool failed: %w", err)
	}

	// Ping immediately so a misconfigured DATABASE_URL fails at startup,
	// not on the first request.
	if err := pool.Ping(c); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database to confirm connection: %w", err)
	}

	return pool, nil
}
