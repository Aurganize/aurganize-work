package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

// RefreshTokenBytes is the entropy of each refresh token. 32 bytes ≈ 256 bits
// — comfortably more than enough that brute-forcing is impractical.
const RefreshTokenBytes = 32

// GenerateRefreshToken creates a cryptographically random refresh token.
// Returns:
//   - the raw token (return to the client, never persisted as-is)
//   - the sha256 hex digest (persisted in sessions.refresh_token_hash)
//
// The caller stores the hash and ships the raw token to the client over
// HTTPS. On refresh, the client presents the raw token; the server hashes
// and looks it up.
func GenerateRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, RefreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	raw = hex.EncodeToString(buf)
	hash = HashRefreshToken(raw)
	return raw, hash, nil
}

// HashRefreshToken computes the sha256 hex digest of a refresh token.
// Use this both when issuing (to derive what we'll store) and when
// verifying (to derive what to look up).
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ErrInvalidRefreshToken is returned when a presented refresh token doesn't
// match any active session. The HTTP layer maps this to a 401.
var ErrInvalidRefreshToken = errors.New("refresh_token_invalid")
