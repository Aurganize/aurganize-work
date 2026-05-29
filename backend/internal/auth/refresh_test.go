package auth

import (
	"strings"
	"testing"
)

func TestGenerateRefreshToken(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed with exception: %v", err)
	}

	// 32 bytes hex-encoded = 64 chars
	if len(raw) != 64 {
		t.Fatalf("raw refresh length: got %d, expected 64", len(raw))
	}

	// sha256 hex = 64 chars
	if len(hash) != 64 {
		t.Fatalf("hashed refresh length: got %d, expected 64", len(raw))
	}

	// Raw and hash must differ; if they're identical we've leaked.
	if raw == hash {
		t.Fatalf("raw == hash; hasing must produce different output")
	}

	// HashRefreshToken should be deterministic.
	if HashRefreshToken(raw) != hash {
		t.Fatalf("HashRefreshToken is not deterministic, i.e Idempotent")
	}

	// hex chars only
	for _, c := range raw + hash {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("non-hex character found in raw or hashed refresh token, token and hash both should be pure hex")
		}
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := range 100 {
		raw, _, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken[%d]: %v", i, err)
		}

		if _, dup := seen[raw]; dup {
			t.Fatalf("duplicate refresh token at iteration %d: %q", i, raw)
		}
		seen[raw] = struct{}{}
	}
}

func TestHashRefreshToken_KnownVector(t *testing.T) {
	got := HashRefreshToken("hello")
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Fatalf("HashRefreshToken(\"hello\")\n got: %s\n want: %s", got, want)
	}
}
