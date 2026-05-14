-- name: GetTenantByID :one
SELECT *
FROM tenants
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTenantBySlug :one
SELECT *
FROM tenants
WHERE slug = $1 AND deleted_at IS NULL;

-- name: CreateTenant :one
INSERT INTO tenants (name, slug, gstin, pan, state_code, plan)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListTenants :many
SELECT *
FROM tenants
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;