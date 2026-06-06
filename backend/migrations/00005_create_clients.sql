-- +goose Up
-- +goose StatementBegin

CREATE TABLE clients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    contact_name    TEXT,
    contact_email   CITEXT,
    contact_phone   TEXT,
    -- Billing address is structured but variable: line1, line2, city, state,
    -- pincode, country. We store as JSONB so India-specific fields (state code,
    -- pincode format) are flexible without a migration each time.
    billing_address JSONB NOT NULL DEFAULT '{}'::jsonb,
    gstin           TEXT,
    pan             TEXT,
    -- 2-char state code (KL, MH, TN, ...). Drives GST CGST/SGST vs IGST
    -- decision at invoice time (Batch 6). Stored here because it's a fact
    -- about the *client's billing address*, not the tenant.
    state_code      TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

-- A client's name is unique within a tenant. Two tenants can each have
-- a client called "Acme" — that's fine.
CREATE UNIQUE INDEX uq_clients_tenant_name
    ON clients (tenant_id, lower(name))
    WHERE deleted_at IS NULL;

-- The pagination index. Every listing query orders by (created_at DESC, id DESC)
-- within a tenant, so this is the workhorse.
CREATE INDEX idx_clients_tenant_created
    ON clients (tenant_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- GSTIN format check (same as tenants table, copy-pasted intentionally — we
-- want this constraint local to every table that stores a GSTIN).
ALTER TABLE clients ADD CONSTRAINT clients_gstin_format
    CHECK (gstin IS NULL OR gstin ~ '^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$');

-- RLS — the standard policy, identical to every other tenant-scoped table.
ALTER TABLE clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE clients FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON clients
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS clients;
-- +goose StatementEnd