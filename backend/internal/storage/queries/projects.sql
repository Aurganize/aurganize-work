-- name: CreateProject :one
INSERT INTO projects (
    tenant_id, client_id, code, name, description, stage, status,
    owner_user_id, estimated_start_at, estimated_end_at, estimated_budget_paise
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetProject :one
SELECT *
FROM projects
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT *
FROM projects
WHERE deleted_at IS NULL
  AND (sqlc.narg('stage')::project_stage IS NULL OR stage = sqlc.narg('stage')::project_stage)
  AND (sqlc.narg('status')::project_status IS NULL OR status = sqlc.narg('status')::project_status)
  AND (sqlc.narg('client_id')::uuid IS NULL OR client_id = sqlc.narg('client_id')::uuid)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (
    sqlc.narg('after_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('after_created_at')::timestamptz, sqlc.narg('after_id')::uuid)
  )
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: UpdateProject :one
UPDATE projects
SET
    name = COALESCE(sqlc.narg('name')::text, name),
    description = COALESCE(sqlc.narg('description')::text, description),
    -- stage handled separately by TransitionProjectStage (see file 09)
    status = COALESCE(sqlc.narg('status')::project_status, status),
    owner_user_id = COALESCE(sqlc.narg('owner_user_id')::uuid, owner_user_id),
    estimated_start_at = COALESCE(sqlc.narg('estimated_start_at')::timestamptz, estimated_start_at),
    estimated_end_at = COALESCE(sqlc.narg('estimated_end_at')::timestamptz, estimated_end_at),
    actual_start_at = COALESCE(sqlc.narg('actual_start_at')::timestamptz, actual_start_at),
    actual_end_at = COALESCE(sqlc.narg('actual_end_at')::timestamptz, actual_end_at),
    estimated_budget_paise = COALESCE(sqlc.narg('estimated_budget_paise')::bigint, estimated_budget_paise),
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProject :exec
UPDATE projects
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: CountProjectsByOwner :one
SELECT count(*) FROM projects
WHERE owner_user_id = $1 AND deleted_at IS NULL AND status = 'active';

-- name: SetProjectStage :one
-- Stage transitions go through this dedicated query rather than UpdateProject.
-- Wraps in RETURNING so callers can confirm the new row state.
UPDATE projects
SET stage = sqlc.arg('stage')::project_stage,
    updated_at = now()
WHERE id = sqlc.arg('id')::uuid AND deleted_at IS NULL
RETURNING *;