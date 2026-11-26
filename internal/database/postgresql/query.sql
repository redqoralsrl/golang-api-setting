-- 공용 쿼리 (advisory lock). 도메인 쿼리는 domain/<name>/postgresql/query.sql 에 둔다.

-- name: TryAdvisoryXactLock :one
select pg_try_advisory_xact_lock(sqlc.arg(ns)::int4, sqlc.arg(key)::int4)::bool;

-- name: TryAdvisoryLock :one
select pg_try_advisory_lock(sqlc.arg(ns)::int4, sqlc.arg(key)::int4)::bool;

-- name: AdvisoryUnlock :one
select pg_advisory_unlock(sqlc.arg(ns)::int4, sqlc.arg(key)::int4)::bool;
