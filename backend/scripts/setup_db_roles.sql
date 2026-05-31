-- Idempotent role setup for Aurganize Work.
-- Run with:
--   psql "$DATABASE_OWNER_URL" \
--     --variable=app_pw="$APP_DB_PASSWORD" \
--     --variable=auth_pw="$AUTH_DB_PASSWORD" \
--     -f backend/scripts/setup_db_roles.sql
--
-- DATABASE_OWNER_URL must connect as the role that owns the schema (in
-- local dev: aurganize). In production this is whichever role your
-- managed Postgres (Neon, RDS) hands you with CREATEROLE privilege.

\set ON_ERROR_STOP on

-- 1) aurganize_app: standard RLS-respecting application role.
--    No BYPASSRLS, no SUPERUSER, no schema-modifying privileges.

SELECT NOT EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = 'aurganize_app'
) AS app_missing \gset

\if :app_missing
    CREATE ROLE aurganize_app LOGIN PASSWORD :'app_pw'
        NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
\else
    ALTER ROLE aurganize_app WITH LOGIN PASSWORD :'app_pw'
        NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
\endif

-- 2) aurganize_auth: read-only role with BYPASSRLS for the narrow set of
--    cross-tenant lookups in AuthService. Note BYPASSRLS — that's the
--    whole point of this role.

SELECT NOT EXISTS (
    SELECT 1 FROM pg_roles WHERE rolname = 'aurganize_auth'
) AS auth_missing \gset

\if :auth_missing
    CREATE ROLE aurganize_auth LOGIN PASSWORD :'auth_pw'
        NOSUPERUSER BYPASSRLS NOCREATEDB NOCREATEROLE;
\else
    ALTER ROLE aurganize_auth WITH LOGIN PASSWORD :'auth_pw'
        NOSUPERUSER BYPASSRLS NOCREATEDB NOCREATEROLE;
\endif

-- 3) Grants for aurganize_app: full DML on current and future tables.

GRANT USAGE ON SCHEMA public TO aurganize_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO aurganize_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO aurganize_app;

-- ALTER DEFAULT PRIVILEGES applies only to objects created by the role
-- that issues this statement. Since migrations run as the owner role
-- (aurganize), this covers every future table created by goose.
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO aurganize_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT ON SEQUENCES TO aurganize_app;

-- 4) Grants for aurganize_auth: SELECT only, no sequences, no future DML.

GRANT USAGE ON SCHEMA public TO aurganize_auth;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO aurganize_auth;

ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO aurganize_auth;

-- 5) Sanity print so a human running this can confirm.

SELECT rolname, rolsuper, rolbypassrls, rolcanlogin
FROM pg_roles
WHERE rolname IN ('aurganize', 'aurganize_app', 'aurganize_auth')
ORDER BY rolname;