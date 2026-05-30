package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// === Request ID ===

// SetRequestID stores the request ID on both gin and context. Called by the
// requestID middleware.
func SetRequestID(c *gin.Context, id string) {
	c.Set(keyRequestID.String(), id)
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), keyRequestID, id),
	)
}

// GetRequestID reads the request ID using gin.Context.Value().
// Value() checks gin storage then request context; since we lookup
// with the typed key, this resolves from c.Request.Context()
// (context.WithValue), not gin's string-keyed c.Set()/c.Get().
// retrieves the request ID, or "" if missing.
func GetRequestID(c context.Context) string {
	if v, ok := c.Value(keyRequestID).(string); ok {
		return v
	}
	return ""
}

// === Auth Context ===

// AuthCtx is everything we know about an authenticated caller. Populated
// by the Auth middleware after JWT validation; consumed by handlers and
// downstream middlewares.
type AuthContext struct {
	TenantId uuid.UUID
	UserId   uuid.UUID
	Role     string
	Client   string // "web" or "mobile"
}

// setAuthCtx stores the auth context on both gin and context for the current request.
func setAuthCtx(c *gin.Context, a *AuthContext) {
	c.Set(keyAuthCtx.String(), a)
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), keyAuthCtx, a),
	)
}

// GetAuthCtx reads the *AuthContext using gin.Context.Value().
// Value() checks gin storage then request context; since we lookup
// with the typed key, this resolves from c.Request.Context()
// (context.WithValue), not gin's string-keyed c.Set()/c.Get().
// retrieves the *AuthContext, or nil if missing.
// GetAuthCtx retrieves the auth context. Returns nil if the request is unauthenticated.
func GetAuthCtx(c context.Context) *AuthContext {
	if v, ok := c.Value(keyAuthCtx).(*AuthContext); ok {
		return v
	}
	return nil
}

// MustAuth retrieves the auth context or panics. Use only in code paths
// guaranteed to be behind the Auth middleware — e.g., a protected handler.
// The Auth middleware itself rejects unauthenticated requests, so reaching
// a "must auth" handler without an AuthCtx is a programming bug, not a
// user error.
func MustAuth(c context.Context) *AuthContext {
	autCtx := GetAuthCtx(c)
	if autCtx == nil {
		panic("MustAuth called outside the Auth middleware")
	}

	return autCtx
}

// === DB connection ===

// SetDBConn stores the tenant-scoped database connection (already inside a
// transaction with SET LOCAL app.tenant_id applied) on the request context.
// Handlers retrieve it via DBConn(ctx) and pass to sqlc queries.
func SetDBConn(c *gin.Context, conn *pgxpool.Pool) {
	c.Set(keyDBConn.String(), conn)
	c.Request = c.Request.WithContext(
		context.WithValue(c.Request.Context(), keyDBConn, conn),
	)
}

// GetDBConn reads the *pgxpool.Conn using gin.Context.Value().
// Value() checks gin storage then request context; since we lookup
// with the typed key, this resolves from c.Request.Context()
// (context.WithValue), not gin's string-keyed c.Set()/c.Get().
// retrieves the *pgxpool.Conn, or nil if missing.
// GetDBConn retrieves the request's tenant-scoped DB connection.
// Returns nil if the request isn't behind the Tenancy middleware.
func GetDBConn(c context.Context) *pgxpool.Conn {
	if v, ok := c.Value(keyDBConn).(*pgxpool.Conn); ok {
		return v
	}
	return nil
}
