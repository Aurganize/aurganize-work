-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- This query runs OUTSIDE of tenant context (during login, before we know
-- which tenant the user belongs to), It's intentionally cross-tenant.
-- Application code MUST use this *only* for login.

-- name: GetUserByEmailAcrossTenants :one
SELECT *
FROM users
WHERE email = $1 AND is_active = true AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByEmailInTenant :one
SELECT *
FROM users
WHERE tenant_id = $1 AND email = $2 AND is_active = true AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = $1;


-- name: UpdateUserPasswordHash :exec
UPDATE users
SET password_hash = $1, updated_at = now()
WHERE id = $2;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false, updated_at = now()
WHERE id = $1;

