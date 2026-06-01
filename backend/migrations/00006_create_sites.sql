-- +goose Up
-- +goose StatementBegin

CREATE TABLE sites (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Denormalised intentionally: every tenant-scoped table has tenant_id
    -- so the RLS policy is identical everywhere.
    client_id                UUID NOT NULL REFERENCES clients(id) ON DELETE RESTRICT,
    name                     TEXT NOT NULL,
    -- Site address: structured as JSONB. Required (NOT NULL) because a site
    -- without an address is useless to the field team.
    address                  JSONB NOT NULL,
    -- Optional geo; field team uses these for navigation when present.
    latitude                 NUMERIC(9, 6),
    longitude                NUMERIC(9, 6),
    contact_on_site_name     TEXT,
    contact_on_site_phone    TEXT,
    -- Access notes — when can the field team enter? Building security gate?
    -- Lift access codes? Plain text, free-form.
    access_notes             TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ
);

-- ON DELETE RESTRICT on client_id: you can't hard-delete a client that still
-- has sites. In practice we soft-delete only, but RESTRICT is a defensive
-- belt against hard-deletes via psql.

-- A site's name is unique within a client. "Bangalore HQ" can exist under
-- Acme and under XYZ Corp — that's two different sites.
CREATE UNIQUE INDEX uq_sites_client_name
    ON sites (tenant_id, client_id, lower(name))
    WHERE deleted_at IS NULL;

-- Listing by client (the most common access pattern: "show me all sites
-- for this client").
CREATE INDEX idx_sites_tenant_client_created
    ON sites (tenant_id, client_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Plain tenant-scoped pagination, for the "all sites across all clients"
-- view in the global Sites list.
CREATE INDEX idx_sites_tenant_created
    ON sites (tenant_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- Lat/lng sanity. Reject nonsensical coordinates that would silently mark
-- a site in the Pacific Ocean.
ALTER TABLE sites ADD CONSTRAINT sites_lat_range
    CHECK (latitude IS NULL OR (latitude BETWEEN -90 AND 90));
ALTER TABLE sites ADD CONSTRAINT sites_lng_range
    CHECK (longitude IS NULL OR (longitude BETWEEN -180 AND 180));

-- Either both lat and lng, or neither. Half-coords are always a bug.
ALTER TABLE sites ADD CONSTRAINT sites_lat_lng_together
    CHECK ((latitude IS NULL) = (longitude IS NULL));

ALTER TABLE sites FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON sites
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sites;
-- +goose StatementEnd