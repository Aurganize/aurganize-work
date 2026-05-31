#!/usr/bin/env bash
# Set up aurganize_app and aurganize_auth roles in the database referenced
# by DATABASE_OWNER_URL. Safe to re-run; rotates passwords if the env vars
# have changed since last run.

set -euo pipefail

: "${DATABASE_OWNER_URL:?DATABASE_OWNER_URL must be set (owner/superuser DSN)}"
: "${APP_DB_PASSWORD:?APP_DB_PASSWORD must be set}"
: "${AUTH_DB_PASSWORD:?AUTH_DB_PASSWORD must be set}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

psql "$DATABASE_OWNER_URL" \
    --variable=app_pw="$APP_DB_PASSWORD" \
    --variable=auth_pw="$AUTH_DB_PASSWORD" \
    --no-psqlrc \
    -f "$SCRIPT_DIR/setup_db_roles.sql"

echo
echo "✓ Roles configured. Verify the printed table above shows:"
echo "    aurganize_app    | f | f | t   (no super, no bypass, can login)"
echo "    aurganize_auth   | f | t | t   (no super, bypass, can login)"