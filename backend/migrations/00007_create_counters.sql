-- +goose Up
-- +goose StatementBegin

-- counters table powers gapless per-tenant per-year numbering for
-- projects, invoices, and any other document type that needs it.
--
-- One row per (tenant_id, scope, year). next_value is incremented
-- under a FOR UPDATE lock, so concurrent allocations are serialised
-- and rollback-safe.
CREATE TABLE counters (
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    scope       TEXT NOT NULL,    -- 'project', 'invoice', ...
    year        INTEGER NOT NULL, -- e.g. 2026
    next_value  BIGINT NOT NULL DEFAULT 1,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, scope, year)
);

-- Scope is a finite enum-ish field; constraint it to known values
-- so a typo in application code doesn't silently allocate a new sequence.
ALTER TABLE counters ADD CONSTRAINT counters_scope_known
    CHECK (scope IN ('project', 'invoice', 'quote'));

ALTER TABLE counters ADD CONSTRAINT counters_year_range
    CHECK (year BETWEEN 2025 AND 2100);

ALTER TABLE counters ENABLE ROW LEVEL SECURITY;
ALTER TABLE counters FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON counters
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS counters;
-- +goose StatementEnd