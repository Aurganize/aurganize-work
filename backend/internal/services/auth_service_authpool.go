package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/jackc/pgx/v5"
)

// === The two cross-tenant lookups ===
//
// Both run on `s.authPool` (aurganize_auth, BYPASSRLS, read-only) in
// READ ONLY transactions. They are the only place in the codebase that
// queries across tenants. Every caller MUST switch to s.appPool for
// any subsequent write.

// findUserByEmail looks up an active user by email across all tenants.
// Used during login (we don't know the tenant yet). Returns
// domain.ErrUnauthenticated for "not found" to preserve the existing
// account-enumeration-resistant error message.
func (s *AuthService) findUserByEmail(ctx context.Context, email string) (gen.Users, error) {
	conn, err := s.authPool.Acquire(ctx)
	if err != nil {
		return gen.Users{}, domain.ErrInternal(err)
	}

	defer conn.Release()

	// ReadOnly tx: belt-and-braces. The role has no INSERT/UPDATE/DELETE
	// grants, so a mutation attempt fails at the database level too.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return gen.Users{}, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)

	querier := gen.New(tx)
	user, err := querier.GetUserByEmailAcrossTenants(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.Users{}, domain.ErrUnauthenticated("invalid credentials", err)
		}
		return gen.Users{}, domain.ErrInternal(err)
	}

	return user, nil
}

// findSessionByHash looks up a non-revoked, non-expired session by its
// refresh-token hash across all tenants. Used during refresh and logout
// (we know only the token, not the tenant). The existing sqlc query
// filters out revoked/expired rows; absence is therefore treated as
// "invalid refresh token".
func (s *AuthService) findSessionByHash(ctx context.Context, tokenHash string) (gen.Sessions, error) {
	conn, err := s.authPool.Acquire(ctx)
	if err != nil {
		return gen.Sessions{}, domain.ErrInternal(err)
	}

	defer conn.Release()

	// ReadOnly tx: belt-and-braces. The role has no INSERT/UPDATE/DELETE
	// grants, so a mutation attempt fails at the database level too.
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return gen.Sessions{}, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)

	querier := gen.New(tx)
	session, err := querier.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.Sessions{}, domain.ErrUnauthenticated("invalid credentials", nil)
		}
		return gen.Sessions{}, domain.ErrInternal(err)
	}

	return session, nil
}
