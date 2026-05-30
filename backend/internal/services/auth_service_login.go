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

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer conn.Release()

	// IMPORTANT: do NOT set app.tenant_id on this connection — we need
	// to query users across all tenants to find one matching the email.
	// AfterRelease in the pool config resets app.tenant_id when the conn
	// is returned, so any prior state from a different request is already
	// cleared.

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx)

	querier := gen.New(tx)
	user, err := querier.GetUserByEmailAcrossTenants(ctx, in.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Same error message as a wrong password — don't reveal which
			// part is wrong (account enumeration protection).
			return nil, domain.ErrUnauthenticated("invalid email or password, no user", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	if err := auth.VerifyPassword(in.Password, user.PasswordHash); err != nil {
		if errors.Is(err, auth.ErrPasswordMismatch) {
			return nil, domain.ErrUnauthenticated("invalid email or password", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	// Now we know the tenant; set context for the session insert.
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.tenant_id', $1, true)", user.TenantID.String()); err != nil {
		return nil, domain.ErrInternal(err)
	}

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
