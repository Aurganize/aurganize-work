-- +goose Up
-- +goose StatementBegin

-- A Session represents one logged-in device. The refresh token is stored
-- hashed (sha256 — refresh tokens are random bytes, no need for bcrypt's
-- slowness). Revoking a session sets revoked_at.
CREATE TABLE sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE,
    client_type         TEXT NOT NULL CHECK (client_type IN ('web', 'mobile')),
    user_agent          TEXT,
    ip_address          INET,
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user_active
    ON sessions (user_id)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_sessions_expires_at
    ON sessions (expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_sessions_token_hash ON sessions (refresh_token_hash);

ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sessions
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd