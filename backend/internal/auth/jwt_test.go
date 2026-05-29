package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestJWT() *JWTService {
	return NewJWTservice("a-secret-that-is-long-enough-32-bytes!!", time.Hour, 24*time.Hour)
}

func TestGenerateAndParse_Web(t *testing.T) {
	jwtSvc := newTestJWT()
	tid, uid := uuid.New(), uuid.New()
	tok, err := jwtSvc.GenerateAccessToken(tid, uid, "admin", ClientWeb)
	if err != nil {
		t.Fatalf("Failed to genereate access token: %v", err)
	}

	claims, err := jwtSvc.ParseAccessToken(tok)
	if err != nil {
		t.Fatalf("Failed to parse access token: %v", err)
	}

	if claims.TenantId != tid {
		t.Fatalf("TenantID: got %v, want %v", claims.TenantId, tid)
	}

	if claims.UserId != uid {
		t.Fatalf("UserID: got %v, want %v", claims.UserId, uid)
	}

	if claims.Role != "admin" {
		t.Fatalf("Role: got %v, want admin", claims.Role)
	}

	if claims.Client != ClientWeb {
		t.Fatalf("Client: got %v, want web", claims.Client)
	}
}

func TestGenerateAndParse_Mobile(t *testing.T) {
	jwtSvc := newTestJWT()
	tok, err := jwtSvc.GenerateAccessToken(uuid.New(), uuid.New(), "pm", ClientMobile)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	claims, err := jwtSvc.ParseAccessToken(tok)
	if claims.Client != ClientMobile {
		t.Fatalf("Client got %v, want mobile", claims.Client)
	}
}

func TestParse_WrongSecret(t *testing.T) {
	a := NewJWTservice("secret-one-that-is-long-enough-aaaa", time.Hour, 24*time.Hour)
	b := NewJWTservice("secret-two-that-is-long-enough-bbbb", time.Hour, 24*time.Hour)
	tok, _ := a.GenerateAccessToken(uuid.New(), uuid.New(), "admin", ClientWeb)
	_, err := b.ParseAccessToken(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParse_Expired(t *testing.T) {
	jwtSvc := newTestJWT()

	jwtSvc.clock = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	tok, _ := jwtSvc.GenerateAccessToken(uuid.New(), uuid.New(), "admin", ClientWeb)
	jwtSvc.clock = time.Now
	_, err := jwtSvc.ParseAccessToken(tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestParse_Tampered(t *testing.T) {
	jwtSvc := newTestJWT()
	tok, _ := jwtSvc.GenerateAccessToken(uuid.New(), uuid.New(), "admin", ClientWeb)
	tampered_tok := tok[:len(tok)-4] + "xxxx"
	_, err := jwtSvc.ParseAccessToken(tampered_tok)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for expired token, got %v", err)
	}
}

func TestParse_AlgNone_Rejected(t *testing.T) {
	const noneToken = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJoYWNrZWQifQ."
	jwtSvc := newTestJWT()
	_, err := jwtSvc.ParseAccessToken(noneToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("alg=none must be rejected, got %v", err)
	}
}
