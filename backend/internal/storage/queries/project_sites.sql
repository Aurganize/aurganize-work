-- name: AddProjectSite :exec
INSERT INTO project_sites (project_id, site_id, tenant_id, notes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (project_id, site_id) DO UPDATE
SET notes = EXCLUDED.notes;

-- name: RemoveProjectSite :exec
DELETE FROM project_sites
WHERE project_id = $1 AND site_id = $2;

-- name: RemoveAllProjectSites :exec
DELETE FROM project_sites
WHERE project_id = $1;

-- name: ListSitesByProject :many
SELECT s.*
FROM sites s
JOIN project_sites ps ON ps.site_id = s.id
WHERE ps.project_id = $1
  AND s.deleted_at IS NULL
ORDER BY s.name;

-- name: ListProjectsBySite :many
SELECT p.*
FROM projects p
JOIN project_sites ps ON ps.project_id = p.id
WHERE ps.site_id = $1
  AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;