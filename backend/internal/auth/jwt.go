package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer identifies tokens issued by this codebase. ParseAccessToken rejects
// tokens whose iss claim doesn't match — protects against token confusion
// across environments (a staging token can't be used in prod and vice versa).
const Issuer = "aurganize-org"

// ClientType is the audience kind — web or mobile. Encoded in the token so
// downstream code (refresh logic, telemetry) doesn't need to re-derive it.
type ClientType string

const (
	ClientWeb    ClientType = "web"
	ClientMobile ClientType = "mobile"
)

// ToString(), stringify the client type
func (c ClientType) ToString() string {
	switch c {
	case ClientWeb:
		return "web"
	case ClientMobile:
		return "mobile"
	default:
		return "unknown platform"
	}
}

// IsValid reports whether the value is one of the known client types.
func (c ClientType) IsValid() bool {
	return (c == "web" || c == "mobile")
}

// AccessClaims is the typed payload of an access JWT.
// jwt.RegisteredClaims provides iss, sub, exp, iat, nbf, aud, jti.
type AccessClaims struct {
	TenantId uuid.UUID  `json:"tid"`
	UserId   uuid.UUID  `json:"sub"`
	Role     string     `json:"role"`
	Client   ClientType `json:"client"`
	jwt.RegisteredClaims
}

// JWTService issues and parses access tokens. It is constructed once with
// the secret + TTLs and reused for the life of the process.
type JWTService struct {
	secret    []byte
	ttlWeb    time.Duration
	ttlMobile time.Duration
	clock     func() time.Time // injectable for tests
}

// NewJWTService wires a JWTService from the application config.
func NewJWTservice(secret string, ttlWeb time.Duration, ttlMobile time.Duration) *JWTService {
	return &JWTService{
		secret:    []byte(secret),
		ttlWeb:    ttlWeb,
		ttlMobile: ttlMobile,
		clock:     time.Now,
	}
}

// GenerateAccessToken builds and signs an access JWT for the given user.
// The TTL depends on the client type (web 8h, mobile 24h by default).
func (s *JWTService) GenerateAccessToken(
	tenantId uuid.UUID,
	userId uuid.UUID,
	role string,
	client ClientType,
) (string, error) {
	if !client.IsValid() {
		return "", fmt.Errorf("invalid client type: %q", client)
	}

	ttl := s.ttlWeb
	if client == ClientMobile {
		ttl = s.ttlMobile
	}

	now := s.clock()
	claims := AccessClaims{
		TenantId: tenantId,
		UserId:   userId,
		Role:     role,
		Client:   client,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userId.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token : %w", err)
	}
	return signed, nil
}

// ErrInvalidToken is returned when a token is malformed, expired, or signed
// with the wrong key. The HTTP layer maps this to a 401.
var ErrInvalidToken = errors.New("invalid token")

// ParseAccessToken verifies the signature, expiry, and issuer of a token
// and returns the typed claims. The error is intentionally opaque: a wrong
// signature, an expired token, and a tampered payload all return ErrInvalidToken,
// so an attacker can't distinguish between them via timing or messages.
func (s *JWTService) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{},
		func(t *jwt.Token) (any, error) {
			// Reject any algorithm other than HS256. Without this check,
			// an attacker could forge a token with alg=none.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
	)

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
