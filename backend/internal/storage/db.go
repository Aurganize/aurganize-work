package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool constructs the application connection pool (aurganize_app).
// RLS-respecting; used by the Tenancy middleware and every handler.
//
// AfterRelease resets app.tenant_id when a connection returns to the pool,
// as a defence-in-depth backup to SET LOCAL inside transactions.
func NewPool(ctx context.Context, databaseUrl string) (*pgxpool.Pool, error) {
	return newPool(ctx, databaseUrl, 25, 2, true)
}

// NewAuthPool constructs the auth pool (aurganize_auth). BYPASSRLS,
// read-only, used by AuthService for cross-tenant lookups only.
//
// Sized small (4/1) because it serves only login/refresh/logout, all of
// which release the connection in milliseconds.
//
// AfterRelease is unnecessary here: this role has BYPASSRLS, so
// app.tenant_id is ignored by Postgres. But we still reset it for
// hygiene — leaving session-local GUCs set across borrows is a code
// smell even when functionally harmless.
func NewAuthPool(ctx context.Context, databaseUrl string) (*pgxpool.Pool, error) {
	return newPool(ctx, databaseUrl, 4, 1, true)
}

// NewPool constructs a pgxpool.Pool from a Postgres URL string.
// Configures sensible defaults for a SaaS workload: 25 max conns,
// 5-minute idle timeout, 1-hour max conn lifetime.
//
// Call once in main(); pass the *pgxpool.Pool by dependency injection
// into handlers and services. Close it in main() via defer.
func newPool(c context.Context, databaseURL string, maxConns, minConns int32, resetTenant bool) (*pgxpool.Pool, error) {
	dbConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Pool sizing — start small, scale by observing.
	// 25 conns × N pods should comfortably handle pilot traffic; Neon's
	// Launch tier supports ~100 connections, so 25 leaves headroom.
	dbConfig.MaxConns = maxConns
	dbConfig.MinConns = minConns
	dbConfig.MaxConnIdleTime = 5 * time.Minute
	dbConfig.MaxConnLifetime = 1 * time.Hour
	dbConfig.HealthCheckPeriod = 30 * time.Second

	// AfterRelease runs every time a connection is returned to the pool.
	// We reset app.tenant_id so the connection is clean for whoever
	// borrows it next. Defence in depth — middleware should already
	// have used SET LOCAL inside a transaction (which auto-resets).
	if resetTenant {
		dbConfig.AfterRelease = func(conn *pgx.Conn) bool {
			_, err := conn.Exec(context.Background(), "SELECT set_config('app.tenant_id','' , false)")
			if err != nil {
				// If we can't reset, drop the connection rather than reuse it.
				return false
			}
			return true
		}
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

	var rolName string
	var bypassRLS bool
	err = pool.QueryRow(c, "SELECT current_user, rolbypassrls FROM pg_roles WHERE rolname = current_user").Scan(&rolName, &bypassRLS)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to check role: %w", err)
	}

	if rolName == "aurganize" || (bypassRLS && rolName != "aurganize_auth") {
		pool.Close()
		return nil, fmt.Errorf(
			"app pool must NOT connect as owner / role with BYPASSRLS, got role = %s, bypassrls = %v",
			rolName, bypassRLS)
	}

	return pool, nil
}
