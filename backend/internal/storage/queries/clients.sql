-- name: CreateClient :one
INSERT INTO clients (
    tenant_id, name, contact_name, contact_email, contact_phone,
    billing_address, gstin, pan, state_code, notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetClient :one
SELECT *
FROM clients
WHERE id = $1 AND deleted_at IS NULL;

-- ListClients fetches a page of clients ordered by (created_at DESC, id DESC).
-- The cursor parameters are (after_created_at, after_id). On the first page
-- both are NULL — the WHERE clause then degenerates to `WHERE TRUE`, which
-- Postgres optimises away via the index.
--
-- Page size is enforced by LIMIT $3. The application caps it at 100.
--
-- name: ListClients :many
SELECT *
FROM clients
WHERE deleted_at IS NULL
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: ListClientsByQuery :many
-- Same as ListClients but with a case-insensitive name search.
SELECT *
FROM clients
WHERE deleted_at IS NULL
  AND name ILIKE '%' || sqlc.arg('query')::text || '%'
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: UpdateClient :one
UPDATE clients
SET
    name = COALESCE(sqlc.narg('name')::text, name),
    contact_name = COALESCE(sqlc.narg('contact_name')::text, contact_name),
    contact_email = COALESCE(sqlc.narg('contact_email')::citext, contact_email),
    contact_phone = COALESCE(sqlc.narg('contact_phone')::text, contact_phone),
    billing_address = COALESCE(sqlc.narg('billing_address')::jsonb, billing_address),
    gstin = COALESCE(sqlc.narg('gstin')::text, gstin),
    pan = COALESCE(sqlc.narg('pan')::text, pan),
    state_code = COALESCE(sqlc.narg('state_code')::text, state_code),
    notes = COALESCE(sqlc.narg('notes')::text, notes),
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteClient :exec
UPDATE clients
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountClients :one
SELECT count(*) FROM clients WHERE deleted_at IS NULL;