package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/config"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Helpers for tests. Spin up a router pointing at the test DB (5433).
func setupTestRouter(t *testing.T) (http.Handler, *auth.JWTService) {
	t.Helper()

	testDb := os.Getenv("DATABASE_URL_TEST")
	if testDb == "" {
		testDb = "postgres://aurganize:aurganize@localhost:5433/aurganize_test?sslmode=disable"
	}
	t.Logf("DATABASE_URL_TEST=%q", os.Getenv("DATABASE_URL_TEST"))
	t.Logf("Using DB=%q", testDb)

	cfg := &config.Config{
		AppEnv:                  "test",
		AppPort:                 "0",
		DatabaseUrl:             testDb,
		JWTSecret:               "a-secret-that-is-very-long-and-good-enough-32",
		JWTAccessTokenTTLWeb:    time.Hour,
		JWTAccessTokneTTLMobile: 24 * time.Hour,
		CORSAllowedOrigins:      []string{"http://localhost:3000"},
		LogLevel:                "info", // quiet in tests
		LogFormat:               "json",
	}

	appPool, err := pgxpool.New(context.Background(), cfg.DatabaseUrl)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	t.Cleanup(appPool.Close)

	authPool, err := pgxpool.New(context.Background(), cfg.DatabaseUrl)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	t.Cleanup(authPool.Close)

	logger := logger.New(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	jwtSvc := auth.NewJWTservice(cfg.JWTSecret, cfg.JWTAccessTokenTTLWeb, cfg.JWTAccessTokneTTLMobile)

	return buildRouter(cfg, logger, appPool, authPool), jwtSvc
}

func TestHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	router, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health test faild : got %d, want 200", rec.Code)
	}
}

func TestProtectedRoute_NoToken(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	router, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token, got %d; want 401", rec.Code)
	}
}

func TestProtectedRoute_BadToken(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	router, _ := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: got %d, want 401", rec.Code)
	}
}

func TestProtectedRoute_GoodToken(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	router, jwtSvc := setupTestRouter(t)

	// We need a real tenant row for the Tenancy middleware to set
	// app.tenant_id against — the SET LOCAL itself succeeds for any UUID,
	// but any query the handler makes would only see rows under it.
	// Our /me handler doesn't query the DB, so we can pass any UUID.
	tid, uid := uuid.New(), uuid.New()
	token, err := jwtSvc.GenerateAccessToken(tid, uid, "admin", auth.ClientWeb)
	if err != nil {
		t.Fatalf("failed to issue token: %v", err)
	}
	t.Logf("generated token %v", token)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/me : got %d; want 200, body = %s", rec.Code, rec.Body)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if body["user_id"] != uid.String() {
		t.Fatalf("user_id : got %v; want %v", body["user_id"], uid.String())
	}

	if body["tenant_id"] != tid.String() {
		t.Fatalf("user_id : got %v; want %v", body["tenant_id"], tid.String())
	}
}
