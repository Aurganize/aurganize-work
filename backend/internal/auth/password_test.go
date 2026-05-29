package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPassword_RoundTrip(t *testing.T) {
	const pw = "correct-horse-battery-staple"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("hash should start with $2x or $2a; got %q", hash[:4])
	}

	if err := VerifyPassword(pw, hash); err != nil {
		t.Fatalf("VerifyPassword on correct password failed: %v", err)
	}
}

func TestHashPassword_DifferentEachTime(t *testing.T) {
	const pw = "correct-horse-battery-staple"
	h1, _ := HashPassword(pw)
	h2, _ := HashPassword(pw)
	if h1 == h2 {
		t.Fatalf("hashes should differ across calls; got identical: %q", h1)
	}

	if err := VerifyPassword(pw, h1); err != nil {
		t.Fatalf("VerifyPassword on correct password failed: %v", err)
	}

	if err := VerifyPassword(pw, h2); err != nil {
		t.Fatalf("VerifyPassword on correct password failed: %v", err)
	}
}

func TestHashPassword_Mismatch(t *testing.T) {
	hash, _ := HashPassword("right_password")
	err := VerifyPassword("wrong_password", hash)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := HashPassword("short")
	if err == nil {
		t.Fatalf("expected error for short password")
	}
}

func TestHashPassword_TooLong(t *testing.T) {
	pw := strings.Repeat("a", 73)
	_, err := HashPassword(pw)
	if err == nil {
		t.Fatalf("expected error for long password (i.e password was more than 72 characters long)")
	}
}

func TestHashPassword_CorruptedHash(t *testing.T) {
	err := VerifyPassword("correct-password", "corrupted_hash_this_is_not_real_hash")
	if err == nil {
		t.Fatalf("expected error for corrupted hash")
	}

	if errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("corrupted-hash error should not be ErrPasswordMismatch")
	}
}
