-- +goose Up
-- +goose StatementBegin

-- Many-to-many join: projects <-> sites.
-- A project can cover multiple sites (e.g., regional rollout);
-- a site can be involved in multiple historical projects.
--
-- We carry tenant_id here too — defensive denormalization. RLS doesn't
-- have to chase through join tables, and the standard policy stays
-- identical.
CREATE TABLE project_sites (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    site_id     UUID NOT NULL REFERENCES sites(id) ON DELETE RESTRICT,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- Notes specific to this site within this project. E.g., "ground floor
    -- only" when the site has multiple floors.
    notes       TEXT,
    added_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, site_id)
);

-- Listing sites of a project.
CREATE INDEX idx_project_sites_project ON project_sites (project_id);

-- Listing projects of a site.
CREATE INDEX idx_project_sites_site ON project_sites (tenant_id, site_id);

ALTER TABLE project_sites ENABLE ROW LEVEL SECURITY;
ALTER TABLE project_sites FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON project_sites
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS project_sites;
-- +goose StatementEnd