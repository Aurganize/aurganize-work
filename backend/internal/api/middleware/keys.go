// Package middleware holds the HTTP middleware composed by the router.
// Each middleware is a single-file implementation with its rationale documented inline.
package middleware

// contextKey is a private type so external packages can't accidentally
// collide with our keys. Postgres docs recommend this pattern;
// Go stdlib uses it too.
type contextKey struct{ name string }

// String makes %v formatting human-readable, useful when dumping context.
func (key contextKey) String() string { return "aurganize:middleware:" + key.name }

// Keys exposed for handler use. Names match what they hold; types are
// always concrete (uuid.UUID, *AuthCtx, *pgxpool.Conn), never interface{}.
var (
	keyRequestID = contextKey{name: "request_id"}
	keyAuthCtx   = contextKey{name: "auth_ctx"}
	keyDBConn    = contextKey{name: "db_conn"}
	keyDBtx      = contextKey{name: "db_tx"}
	keyLogger    = contextKey{name: "logger"}
)
