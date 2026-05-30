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

// Refresh consumes a refresh token and issues a new access + refresh pair.
// The old refresh token is replaced (rotation). If the presented token is
// missing, expired, or already-used (replay), we return Unauthenticated.
//
// Replay detection note: when a token doesn't match an active session, we
// can't tell whether it's never existed or has been consumed. In v2 we'll
// keep the rotated-from hash for ~5 minutes and use a match against it as
// the signal to revoke ALL sessions for the user. For Batch 1 we accept
// the simpler semantics (any non-match = 401).
func (s *AuthService) Refresh(ctx context.Context, in RefreshInput) (*TokenPair, error) {
	in.RefreshToken = strings.TrimSpace(in.RefreshToken)
	if in.RefreshToken == "" {
		return nil, domain.ErrUnauthenticated("missing refresh token", nil)
	}

	if !in.Client.IsValid() {
		return nil, domain.ErrInvalidInput("invalid client type", nil)
	}

	hash := auth.HashRefreshToken(in.RefreshToken)

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)

	// We don't know the tenant yet — look up session without tenancy.
	querier := gen.New(tx)

	session, err := querier.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUnauthenticated("invalid or expired refresh token", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	// Set tenancy now that we know it.
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.tenant_id', $1, true)",
		session.TenantID.String()); err != nil {
		return nil, domain.ErrInternal(err)
	}

	user, err := querier.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUnauthenticated("user not found, in tenant context", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	if !user.IsActive {
		return nil, domain.ErrUnauthenticated("user account is disabled", nil)
	}

	// Rotate: revoke the current session and issue a fresh one.
	// (We could update-in-place, but creating a new row makes the audit
	// trail clearer and the implementation simpler.)
	if err := querier.RevokeSession(ctx, session.ID); err != nil {
		return nil, domain.ErrInternal(err)
	}

	tokenPair, err := s.issueSessionTokens(ctx, querier, session.TenantID, user.ID, string(user.Role), in)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return tokenPair, nil
}
