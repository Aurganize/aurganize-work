-- name: CreateSite :one
INSERT INTO sites (
    tenant_id, client_id, name, address, latitude, longitude,
    contact_on_site_name, contact_on_site_phone, access_notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetSite :one
SELECT *
FROM sites
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListSites :many
SELECT *
FROM sites
WHERE deleted_at IS NULL
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: ListSitesByClient :many
SELECT *
FROM sites
WHERE deleted_at IS NULL
  AND client_id = sqlc.arg('client_id')::uuid
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: UpdateSite :one
UPDATE sites
SET
    name = COALESCE(sqlc.narg('name')::text, name),
    address = COALESCE(sqlc.narg('address')::jsonb, address),
    latitude = COALESCE(sqlc.narg('latitude')::numeric, latitude),
    longitude = COALESCE(sqlc.narg('longitude')::numeric, longitude),
    contact_on_site_name = COALESCE(sqlc.narg('contact_on_site_name')::text, contact_on_site_name),
    contact_on_site_phone = COALESCE(sqlc.narg('contact_on_site_phone')::text, contact_on_site_phone),
    access_notes = COALESCE(sqlc.narg('access_notes')::text, access_notes),
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteSite :exec
UPDATE sites
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountSitesByClient :one
SELECT count(*) FROM sites WHERE client_id = $1 AND deleted_at IS NULL;