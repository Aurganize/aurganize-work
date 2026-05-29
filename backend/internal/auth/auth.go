package auth

import "time"

// PrimitiveService bundles the four auth primitives into a single dependency
// the AuthService (file 06) can hold. This is purely organisational — no logic,
// just a container.
type PrimitiveService struct {
	JWT *JWTService
}

// NewPrimitiveService builds the primitive service from config values.
// Called once in main() and injected into the AuthService.
func NewPrimitiveService(jwtSecret string, accessTTLWeb, accessTTLMobile time.Duration) *PrimitiveService {
	return &PrimitiveService{
		JWT: NewJWTservice(jwtSecret, accessTTLWeb, accessTTLMobile),
	}
}

// HashPassword, VerifyPassword, GenerateRefreshToken, HashRefreshToken
// are package-level functions and don't need to live on the struct.
// The AuthService calls them directly via the auth package import.
