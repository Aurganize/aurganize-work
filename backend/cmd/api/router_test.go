package main

import (
	"net/http"
	"os"
	"testing"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/config"
)

func setupTestRouter(t *testing.T) (http.Handler, *auth.JWTService) {
	t.Helper()

	testDb := os.Getenv("DATABASE_URL_TEST")
	if testDb == "" {
		testDb = "postgres://aurganize:aurganize"
	}

	cfg := &config.Config{
		AppEnv:      "test",
		AppPort:     "0",
		DatabaseUrl: test,
	}
}
