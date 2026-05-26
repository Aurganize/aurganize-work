-- name: CreateSession :one
INSERT INTO sessions(
	tenant_id, user_id, refresh_token_hash, 
	client_type,  user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetSessionByTokenHash :one
SELECT * 
FROM sessions
WHERE refresh_token_hash = $1
	AND revoked_at IS NULL
	AND expires_at > now();

-- name: TouchSession :exec
UPDATE sessions
SET last_used_at = now()
WHERE id = $1;

-- name: RotateSessionToken :exec
UPDATE sessions
SET refresh_token_hash = $1,
	last_used_at = now(),
	expires_at = $2
WHERE id = $3;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE sessions
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: PurgeExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < now() - INTERVAL '30 days';