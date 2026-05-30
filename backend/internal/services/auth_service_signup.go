package services

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Signup creates a brand-new tenant and its first admin user, then issues
// a session (access + refresh tokens). This is the only endpoint that
// runs without tenancy context (we're creating the tenant!).
//
// Validation already happened at the handler layer. Here we assert
// business rules and persist atomically:
//
//  1. Open a transaction (no SET LOCAL — tenant doesn't exist yet).
//  2. Insert the tenant.
//  3. SET LOCAL app.tenant_id to the new tenant's ID — RLS now lets us
//     insert the user row.
//  4. Insert the admin user with bcrypt-hashed password.
//  5. Generate refresh token, hash it, insert the session.
//  6. Generate access JWT.
//  7. Commit.
//
// If anything fails between (1) and (7), the transaction rolls back and
// nothing persists. No half-created tenant.
func (s *AuthService) SignUp(ctx context.Context, in SignupInput) (*TokenPair, error) {
	// Trim and lowercase the slug — DB constraint expects lowercase.
	in.TenantSlug = strings.ToLower(strings.TrimSpace(in.TenantSlug))
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if !in.Client.IsValid() {
		return nil, domain.ErrInvalidInput("invalid client type", nil)
	}

	// Hash the password before opening any DB resources. If the password is
	// too long, fail fast before touching the DB.
	passwordHash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, domain.ErrInvalidInput(err.Error(), err)
	}

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	defer tx.Rollback(ctx) // safe - no-op after commit

	querier := gen.New(tx)

	// Step 1: insert tenant
	tenant, err := querier.CreateTenant(ctx, gen.CreateTenantParams{
		Name:      in.TenantName,
		Slug:      in.TenantSlug,
		Gstin:     nil,
		Pan:       nil,
		StateCode: nil,
		Plan:      "pilot",
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return nil, domain.ErrConflict("tenant slug already takem", err)
			case "23514": // check_violation
				return nil, domain.ErrInvalidInput("tenant slug or gstin format invalid", err)
			}
		}
		return nil, domain.ErrInternal(err)
	}

	// Step 2: SET LOCAL app.tenant_id so the user insert satisfies RLS.
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.tenant_id', $1, true)",
		tenant.ID.String()); err != nil {
		return nil, domain.ErrInternal(err)
	}

	// Step 3: insert admin user
	user, err := querier.CreateUser(ctx, gen.CreateUserParams{
		TenantID:     tenant.ID,
		Email:        in.Email,
		PasswordHash: passwordHash,
		Name:         in.AdminName,
		Role:         gen.UserRoleAdmin,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, pgErr) {
			switch pgErr.Code {
			// shouldn't happen — we just created the tenant — but defensive.
			case "23505":
				return nil, domain.ErrInvalidInput("email already in use", err)
			}
		}
		return nil, domain.ErrInternal(err)
	}

	// Step 4: issue tokens (access + refresh) and persist the session.
	tokenPair, err := s.issueSessionTokens(ctx, querier, tenant.ID, user.ID, string(user.Role), in)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal(err)
	}

	return tokenPair, nil
}

// issueSession is the shared helper used by Signup, Login, and Refresh:
//   - Generate access JWT for the user.
//   - Generate random refresh token + its sha256 hash.
//   - Insert a `sessions` row recording the hash, client type, etc.
//   - Return the raw refresh token to the caller (NEVER persisted directly).
//
// Caller is responsible for the transaction lifecycle.
func (s *AuthService) issueSessionTokens(
	ctx context.Context, q *gen.Queries,
	tenantID, userID uuid.UUID, role string,
	meta any, // SignupInput / LoginInput / RefreshInput, used for user-agent + IP
) (*TokenPair, error) {
	clientType, userAgent, ipAddress := extractClientMeta(meta)

	// Choose TTLs by client type.
	refreshTTl := s.refreshTTLWeb
	if clientType == auth.ClientMobile {
		refreshTTl = s.refreshTTLMobile
	}
	rawRefresh, hashedRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	access, err := s.jwt.GenerateAccessToken(tenantID, userID, role, clientType)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	// Build params for session insert.
	expiresAt := time.Now().Add(refreshTTl)
	var ipPtr *netip.Addr
	if ipAddress != "" {
		if address, err := netip.ParseAddr(ipAddress); err != nil {
			ipPtr = &address
		}
	}

	_, err = q.CreateSession(ctx, gen.CreateSessionParams{
		TenantID:         tenantID,
		UserID:           userID,
		RefreshTokenHash: hashedRefresh,
		ClientType:       string(clientType),
		UserAgent:        nullableString(userAgent),
		IpAddress:        ipPtr,
		ExpiresAt:        expiresAt,
	})

	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		// Compute access-token expiry for the client. Sliding refresh aside,
		// the client uses this to schedule refresh just-in-time.
		ExpiresAt: time.Now().Add(s.jwt.GetAccessTTL(clientType)),
		TokenType: "Bearer",
	}, nil

}

// extractClientMeta pulls the client type, user agent, and IP from one of
// the three Input structs. Switch is small; tests cover it.
func extractClientMeta(meta any) (auth.ClientType, string, string) {
	switch m := meta.(type) {
	case SignupInput:
		return m.Client, m.UserAgent, m.IPAddress
	case LoginInput:
		return m.Client, m.UserAgent, m.IPAddress
	case RefreshInput:
		return m.Client, m.UserAgent, m.IPAddress
	}
	return auth.ClientWeb, "", ""
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
