package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config is the single, immutable runtime configuration of the API server.
// Built once in main() via Load(), then passed by value (or pointer) into
// constructors. Never accessed as a global
type Config struct {

	// ====== Application ======

	AppEnv     string `envconfig:"APP_ENV" default:"development"`
	AppPort    string `envconfig:"APP_PORT" default:"8080"`
	AppBaseURL string `envconfig:"APP_BASE_URL" default:"http://localhost:8080"`

	// ====== Database =======

	DatabaseUrl string `envconfig:"DATABASE_URL" required:"true"`

	// ====== JWT / Auth ======

	JWTSecret               string        `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessTokenTTLWeb    time.Duration `envconfig:"JWT_ACCESS_TOKEN_TTL_WEB" default:"8h"`
	JWTAccessTokneTTLMobile time.Duration `envconfig:"JWT_ACCESS_TOKEN_TTL_MOBILE" default:"24h"`
	RefreshTokenTTLWeb      time.Duration `envconfig:"REFRESH_TOKEN_TTL_WEB" default:"168h"`    // 7 days
	RefreshTokenTTLMobile   time.Duration `envconfig:"REFRESH_TOKEN_TTL_MOBILE" default:"720h"` // 30 days

	// ====== CORS ======

	CORSAllowedOrigins []string `envconfig:"CORS_ALLOWED_ORIGINS" default:"http://localhost:3000;http://localhost:5173"`

	// ====== Email (Resend) - Optional in dev, required in prod ======

	ResendAPIkey     string `envconfig:"RESEND_API_KEY"`
	EmailFromAddress string `envconfig:"EMAIL_FROM_ADDRESS" default:"noreply@aurganize.com"`
	EmailFromName    string `envconfig:"EMAIL_FROM_NAME" default:"Aurganize Work"`

	// ===== Blob Storage (Cloudfare R2) - empty in for now, filled later on ======

	R2AccountID       string `envconfig:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `envconfig:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `envconfig:"R2_SECRET_ACCESS_KEY"`
	R2Bucket          string `envconfig:"R2_BUCKET"`
	R2PublicURL       string `envconfig:"R2_PUBLIC_URL"`

	// ====== Logging ======

	LogLevel  string `envconfig:"LOG_LEVEL" default:"debug"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"` // json | text
}

// IsProduction is true when APP_ENV is set to "production". Used to flip
// behaviour like CORS strictness, log format default, and prod-only checks
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppPort, "production")
}

// IsDevelopment is true for local development. Used to enable verbose error
// responses and developer-friendly defaults.
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development")
}

// Load reads configuration from environment variables, validates it,
// and returns either a populated Config or an error explaining what's missing
// Call this once in main(); never read env vars directly elsewhere
func load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// validate enforces runtime invariants beyond what envconfig's required tag
// catches. Anything that depends on other fields, or that needs semantic
// checking (e.g minimum length), goes here
func (c *Config) Validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("invalid JWT_SECRET must be at least 32 characters")
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid LOG_LEVEL %q (must be debug, info, error, warn)", c.LogLevel)
	}

	switch strings.ToLower(c.LogFormat) {
	case "josn", "text":
	default:
		return fmt.Errorf("invalid LOG_FORMAT %q (must be json or text)", c.LogFormat)
	}

	if c.IsProduction() {
		if c.ResendAPIkey == "" {
			return errors.New("RESEND_API_KEY is required in production")
		}

		if c.AppBaseURL == "http://localhost:8080" {
			return errors.New("APP_BASE_URL must be set to the public URL in production")
		}

	}
	return nil
}
