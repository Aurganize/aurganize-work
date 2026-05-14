-- +goose Up
-- +goose StatementBegin

-- pgcrypto gives us gen_random_uuid() for default PKs
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- citext = case-insensitive text. Used for emails.
CREATE EXTENSION IF NOT EXISTS citext;

-- A tenant is one organisation using the platform.
-- The pilot company is tenant #1; the architecture supports many.
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    gstin       TEXT,
    pan         TEXT,
    plan        TEXT NOT NULL DEFAULT 'pilot',
    state_code  TEXT,  -- 2-char state code for GST intra/inter-state (e.g., "KL", "MH")
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Slug must be URL-safe: lowercase, alphanumeric, hyphens only, 3-40 chars
ALTER TABLE tenants ADD CONSTRAINT tenants_slug_format
    CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$');

-- GSTIN format: 2-digit state + 10-char PAN + 1-digit entity + Z + 1 checksum
-- We allow NULL (tenant may not be GST-registered initially) but if provided must match.
ALTER TABLE tenants ADD CONSTRAINT tenants_gstin_format
    CHECK (gstin IS NULL OR gstin ~ '^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$');

CREATE INDEX idx_tenants_slug ON tenants (slug) WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenants;
DROP EXTENSION IF EXISTS citext;
DROP EXTENSION IF EXISTS pgcrypto;
-- +goose StatementEnd