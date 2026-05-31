-- +goose Up
-- +goose StatementBegin

-- Belt-and-braces: ensure RLS applies even to the table owner. Without
-- this, the owner role (aurganize) silently bypasses every policy. With
-- it, even a misconfigured connection string back to the owner cannot
-- leak cross-tenant data.
--
-- Future tenant-scoped table migrations MUST include the same statement
-- alongside ALTER TABLE … ENABLE ROW LEVEL SECURITY. See the patch doc
-- (06_5) section "What every future migration must include".

ALTER TABLE users    FORCE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users    NO FORCE ROW LEVEL SECURITY;
-- +goose StatementEnd