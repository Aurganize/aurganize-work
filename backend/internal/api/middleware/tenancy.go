package middleware

import (
	"log/slog"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenancy returns a middleware that sets up tenant-scoped database access
// for the request. Steps:
//
//  1. Read the AuthCtx (must be present — Tenancy goes after Auth).
//  2. Acquire a connection from the pool.
//  3. Begin a transaction.
//  4. Run `SELECT set_config('app.tenant_id', $1, true)`. The `true` arg
//     scopes the setting to the current transaction. Postgres RLS now sees
//     it.
//  5. Stash the connection on the context for the handler.
//  6. After the handler runs, commit if everything was fine, rollback
//     otherwise, then release the connection.
//
// Handlers MUST run their queries on `middleware.DBConn(ctx)`, not on
// `pool.Acquire(ctx)`. The former is RLS-scoped; the latter is not.
func Tenancy(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		logger := logger.FromContext(ctx.Request.Context())
		authCtx := GetAuthCtx(ctx.Request.Context())

		if authCtx == nil {
			// Programming error — Tenancy must be chained after AuthRequired.
			logger.Error("Tenancy Middleware reached with a valid Auth Context set")
			response.RenderError(ctx, domain.ErrInternal(nil))
			return
		}

		conn, err := pool.Acquire(ctx)
		if err != nil {
			logger.Error("acquired Db connection failed:", slog.Any("err", err))
			response.RenderError(ctx, domain.ErrInternal(err))
			return
		}

		tx, err := conn.BeginTx(ctx.Request.Context(), pgx.TxOptions{})
		if err != nil {
			conn.Release()
			logger.Error("request scopped transaction creation failed", slog.Any("err", err))
			response.RenderError(ctx, domain.ErrInternal(err))
			return
		}

		// SET LOCAL is the right tool: it's transaction-scoped, so on COMMIT
		// or ROLLBACK the setting is automatically cleared. Even if the
		// connection is reused for a different tenant later, no leak.
		_, err = tx.Exec(
			ctx.Request.Context(),
			"SELECT set_config('app.tenant_id', $1, true)",
			authCtx.TenantId.String(),
		)

		if err != nil {
			_ = tx.Rollback(ctx.Request.Context())
			conn.Release()
			logger.Error("failed to set tenant context to db connection", slog.Any("err", err))
			response.RenderError(ctx, domain.ErrInternal(err))
			return
		}

		// Stash the transaction-as-pgx.Tx on the context. Queries use this.
		// We use a tx (not the conn) so all queries in the request commit/rollback together.
		SetDBtx(ctx, tx)
		SetDBConn(ctx, conn)

		// Run the handler.
		ctx.Next()

		// Decide commit vs rollback based on the response status.
		// 5xx → rollback; others → commit. 4xx errors that ran successful
		// reads might write audit-log rows we want to keep; we'd revisit
		// this if it matters. For Batch 1, 4xx = rollback to be conservative.
		status := ctx.Writer.Status()
		if status >= 500 || status >= 400 && len(ctx.Errors) > 0 {
			if err := tx.Rollback(ctx.Request.Context()); err != nil {
				logger.Error("failed to rollback transaction with errors", slog.Any("err", err))
			}
		} else {
			if err := tx.Commit(ctx.Request.Context()); err != nil {
				logger.Error("failed to commited transaction", slog.String("request-id", GetRequestID(ctx.Request.Context())))
				// Too late to send a 500; the headers are already written.
				// But the rollback path of the deferred release is still
				// correct, and downstream observability will catch it.
			}
		}

		conn.Release()

	}
}
