package services

import (
	"context"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AuthService orchestrates the auth primitives + storage queries.
// One instance per process; safe for concurrent use.
type AuthService struct {
	jwt              *auth.JWTService
	appPool          DBPool
	authPool         DBPool
	refreshTTLWeb    time.Duration
	refreshTTLMobile time.Duration
}

// DBPool is the minimal interface the AuthService needs from pgxpool.
// Defining it as an interface lets unit tests pass a mock.
type DBPool interface {
	Acquire(ctx context.Context) (PoolConn, error)
}

// PoolConn is the minimal interface acquired from a DBPool.
type PoolConn interface {
	Release()
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

// NewAuthService constructs an AuthService. Called once in main(), injected
// into the auth handler.
func NewAuthService(
	jwt *auth.JWTService,
	appPool DBPool,
	authPool DBPool,
	refreshTTLWeb, refreshTTLMobile time.Duration,
) *AuthService {
	return &AuthService{
		jwt:              jwt,
		appPool:          appPool,
		authPool:         authPool,
		refreshTTLWeb:    refreshTTLWeb,
		refreshTTLMobile: refreshTTLMobile,
	}
}

// === Input/output types ===
//
// We define explicit types here rather than reusing storage.User etc.,
// so the service contract is independent of the database schema. If we
// change the column layout, the service contract is untouched.

type SignupInput struct {
	TenantName string
	TenantSlug string
	AdminName  string
	Email      string
	Password   string
	Client     auth.ClientType
	UserAgent  string
	IPAddress  string
}

type LoginInput struct {
	Email     string
	Password  string
	Client    auth.ClientType
	UserAgent string
	IPAddress string
}

type RefreshInput struct {
	RefreshToken string
	Client       auth.ClientType
	UserAgent    string
	IPAddress    string
}

type LogoutInput struct {
	RefreshToken string
}

// TokenPair is what signup/login/refresh all return.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"` // always "Bearer"
}

// MeResult is the typed result of fetching the current user.
type MeResult struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
}
