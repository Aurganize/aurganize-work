package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the work factor used for password hashing.
// 12 ≈ 250ms on a modern CPU. Increase carefully: each +1 doubles cost.
const BcryptCost = 12

// MinPasswordLength is enforced at the application layer (validator tag).
// Even a perfect hash function can't help with "password123".
const MinPasswordLength = 8

// MaxPasswordLength is enforced at the application layer (validator tag).
const MaxPasswordLength = 72

// ErrPasswordMismatch is returned by Verify when the password is wrong.
// We expose this as a sentinel so callers can distinguish "wrong password"
// from "hash format corrupted" without string matching.
var ErrPasswordMismatch = errors.New("password mismatch")

// HashPassword takes a plaintext password and returns a bcrypt hash safe
// to store in the database. The hash is self-describing (includes algorithm,
// cost, and salt), so future verification doesn't need this constant.
//
// Returns an error only for invariant violations (e.g., password too long;
// bcrypt rejects > 72 bytes). The returned hash is ASCII-safe.
func HashPassword(plaintext string) (string, error) {
	if len(plaintext) < MinPasswordLength {
		return "", fmt.Errorf("password must be atleast %d characters long", MinPasswordLength)
	}

	// bcrypt has a hard upper bound of 72 bytes on input. Passwords longer
	// than that get silently truncated, which is a footgun: two different
	// long passwords could hash identically. Reject explicitly instead.
	if len(plaintext) > MaxPasswordLength {
		return "", fmt.Errorf("password must be at most %d", MaxPasswordLength)
	}

	h, err := bcrypt.GenerateFromPassword([]byte(plaintext), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate hash: %w", err)
	}

	return string(h), nil
}

// VerifyPassword compares a plaintext password against a stored bcrypt hash.
// Returns nil on match, ErrPasswordMismatch on mismatch, or a different
// error if the stored hash is malformed (data corruption — should not occur
// in practice).
//
// Verification is constant-time relative to the hash; the caller does not
// need to worry about timing attacks on the comparison itself.
func VerifyPassword(plaintext, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	if err == nil {
		return nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}
	return fmt.Errorf("failed to verify hash: %w", err)
}
