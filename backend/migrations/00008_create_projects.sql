-- +goose Up
-- +goose StatementBegin

-- Stages are an enum so the database refuses invalid values. Adding a stage
-- in a future migration is a deliberate ALTER TYPE — not a string-typo risk.
CREATE TYPE project_stage AS ENUM (
    'lead',
    'survey',
    'design',
    'quote',
    'production',
    'signoff',
    'completed'
);

-- Statuses orthogonal to stage. A 'paused' project can be at any stage.
CREATE TYPE project_status AS ENUM (
    'active',
    'paused',
    'cancelled'
);

CREATE TABLE projects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id           UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    -- Project code is generated server-side and unique within tenant.
    -- Format: P/YYYY/NNN (e.g., P/2026/001).
    code                TEXT NOT NULL,
    name                TEXT NOT NULL,
    description         TEXT,
    stage               project_stage NOT NULL DEFAULT 'lead',
    status              project_status NOT NULL DEFAULT 'active',
    -- Owner is the PM responsible. Optional at creation; assignable later.
    -- ON DELETE SET NULL: removing the owner user shouldn't orphan the project.
    owner_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    -- Estimated dates set at planning; actual at completion.
    estimated_start_at  TIMESTAMPTZ,
    estimated_end_at    TIMESTAMPTZ,
    actual_start_at     TIMESTAMPTZ,
    actual_end_at       TIMESTAMPTZ,
    -- Budget in paise (smallest INR unit). Avoids float-rounding bugs in money.
    -- NULL means budget not yet set. 0 is explicit "no budget allocated."
    estimated_budget_paise  BIGINT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);

-- Project code is unique within a tenant.
CREATE UNIQUE INDEX uq_projects_tenant_code
    ON projects (tenant_id, code)
    WHERE deleted_at IS NULL;

-- The workhorse listing index.
CREATE INDEX idx_projects_tenant_created
    ON projects (tenant_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Frequently-used filters: "show me all active design-stage projects."
CREATE INDEX idx_projects_tenant_stage
    ON projects (tenant_id, stage)
    WHERE deleted_at IS NULL AND status = 'active';

-- Filtering by owner: "what's on my plate?"
CREATE INDEX idx_projects_owner
    ON projects (tenant_id, owner_user_id)
    WHERE deleted_at IS NULL AND owner_user_id IS NOT NULL;

-- Sanity: if both estimated dates are present, end must be at/after start.
ALTER TABLE projects ADD CONSTRAINT projects_est_dates_ordered
    CHECK (estimated_start_at IS NULL
        OR estimated_end_at IS NULL
        OR estimated_end_at >= estimated_start_at);

-- Same for actuals.
ALTER TABLE projects ADD CONSTRAINT projects_actual_dates_ordered
    CHECK (actual_start_at IS NULL
        OR actual_end_at IS NULL
        OR actual_end_at >= actual_start_at);

-- Budget non-negative.
ALTER TABLE projects ADD CONSTRAINT projects_budget_nonneg
    CHECK (estimated_budget_paise IS NULL OR estimated_budget_paise >= 0);

ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE projects FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON projects
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS projects;
DROP TYPE IF EXISTS project_status;
DROP TYPE IF EXISTS project_stage;
-- +goose StatementEnd