-- +goose Up
-- +goose StatementBegin

-- User roles are enumerated in the database for referential safety.
-- Adding a new role is a deliberate migration, not a string typo.
CREATE TYPE user_role AS ENUM (
    'admin',
    'pm',
    'sales',
    'support',
    'designer',
    'finance',
    'field'
);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email           CITEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    name            TEXT NOT NULL,
    role            user_role NOT NULL DEFAULT 'pm',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    -- A user's email is unique within their tenant. The same email
    -- can exist across tenants (consultant working for two firms).
    UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant_active ON users (tenant_id) WHERE is_active = true AND deleted_at IS NULL;
CREATE INDEX idx_users_email ON users (email) WHERE deleted_at IS NULL;

-- RLS — the standard pattern repeated identically on every tenant-scoped table.
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Policy: a row is visible/modifiable only if the session's app.tenant_id matches.
-- Using FOR ALL covers SELECT, INSERT, UPDATE, DELETE in one policy.
CREATE POLICY tenant_isolation ON users
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd