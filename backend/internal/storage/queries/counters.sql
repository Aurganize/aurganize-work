-- GetCounterForUpdate locks the counter row for the (tenant, scope, year)
-- type under FOR UPDATE. Concurrent callers block here until commit/rollback.]
--
-- name: GetCounterForUpdate :one
SELECT *
FROM counters
WHERE tenant_id =$1 AND scope = $2 AND year = $3
FOR UPDATE;


-- InsertCounterIfMissing creates the counter row when it doesn't exist.
-- ON CONFLICT DO NOTHING because two concurrent allocations might both
-- discover the missing row; one wins the insert, the other proceeds to 
-- SELECT FOR UPDATE.
--
-- name: InsertCounterIfMissing :exec
INSERT INTO counters (tenant_id, scope, year, next_value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, scope, year) DO NOTHING;

-- IncrementCounter advances the next_value by 1. Must run inside the same
-- transaction as the prior GetCounterForUpdate.
--
-- name: IncrementCounter :exec
UPDATE counters
SET next_value = next_value + 1, updated_at = now()
WHERE tenant_id = $1 AND scope = $2 AND year = $3;