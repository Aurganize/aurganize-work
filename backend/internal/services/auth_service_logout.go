package services

import (
	"context"
	"errors"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/jackc/pgx/v5"
)

// Logout revokes the session associated with the presented refresh token.
// Idempotent: re-calling logout with the same (now-invalid) token returns
// success — no client-side complexity.
func (s *AuthService) Logout(ctx context.Context, in LogoutInput) error {
	in.RefreshToken = strings.TrimSpace(in.RefreshToken)
	if in.RefreshToken == "" {
		// Empty token is a no-op — already logged out.
		return nil
	}

	hashedRefresh := auth.HashRefreshToken(in.RefreshToken)

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return domain.ErrInternal(err)
	}

	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)

	querier := gen.New(tx)
	session, err := querier.GetSessionByTokenHash(ctx, hashedRefresh)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already gone; success
		}
		return domain.ErrInternal(err)
	}

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.tenant_id', $1, true)", session.TenantID.String()); err != nil {
		return domain.ErrInternal(err)
	}

	if err := querier.RevokeSession(ctx, session.ID); err != nil {
		return domain.ErrInternal(err)
	}

	return tx.Commit(ctx)
}
