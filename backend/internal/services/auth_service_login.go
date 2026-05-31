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

// Login authenticates with email + password and issues a new session.
//
// The catch with multi-tenancy: at login time we don't know which tenant
// the user belongs to. We resolve by querying users WITHOUT a tenant
// context, using GetUserByEmailAcrossTenants. The query bypasses RLS by
// running on a connection that has NOT had app.tenant_id set.
//
// Once we have the user (and their tenant_id), the rest of the flow runs
// with SET LOCAL applied.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if !in.Client.IsValid() {
		return nil, domain.ErrInvalidInput("invalid client type", nil)
	}

	if in.Email == "" || in.Password == "" {
		return nil, domain.ErrInvalidInput("email and password required", nil)
	}

	// === Stage 1: cross-tenant lookup on authPool ===
	user, err := s.findUserByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Same error message as a wrong password — don't reveal which
			// part is wrong (account enumeration protection).
			return nil, domain.ErrUnauthenticated("invalid email or password", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	// Password check is pure in-memory; no DB resources held.
	if err := auth.VerifyPassword(in.Password, user.PasswordHash); err != nil {
		if errors.Is(err, auth.ErrPasswordMismatch) {
			return nil, domain.ErrUnauthenticated("invalid email or password", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	// === Stage 2: side effects on appPool, RLS-scoped ===
	conn, err := s.appPool.Acquire(ctx)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)
	// Now we know the tenant; set context for the session insert.
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.tenant_id', $1, true)", user.TenantID.String()); err != nil {
		return nil, domain.ErrInternal(err)
	}

	querier := gen.New(tx)

	// Update last_login_at (best-effort; non-fatal).
	querier.UpdateUserLastLogin(ctx, user.ID)

	tokenPair, err := s.issueSessionTokens(ctx, querier, user.TenantID, user.ID, string(user.Role), in)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return tokenPair, nil
}
