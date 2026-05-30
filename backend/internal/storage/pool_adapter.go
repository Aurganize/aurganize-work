package storage

import (
	"context"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolAdapter wraps *pgxpool.Pool to satisfy services.DBPool.
// The wrapper exists so the AuthService can be unit-tested with a mock
// while production uses the real pool.
type PoolAdapter struct {
	Pool *pgxpool.Pool
}

func (p *PoolAdapter) Acquire(ctx context.Context) (services.PoolConn, error) {
	conn, err := p.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	return &connAdapter{
		conn: conn,
	}, nil
}

type connAdapter struct {
	conn *pgxpool.Conn
}

func (c *connAdapter) Release() { c.conn.Release() }

func (c *connAdapter) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) {
	return c.conn.BeginTx(ctx, txOptions)
}

func (c *connAdapter) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return c.conn.Exec(ctx, sql, arguments...)
}

func (c *connAdapter) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return c.conn.QueryRow(ctx, sql, arguments...)
}
