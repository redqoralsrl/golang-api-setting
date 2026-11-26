-- name: CreateUser :one
insert into v1.users (email, password_hash)
values (sqlc.arg(email), sqlc.arg(password_hash))
returning id, created_at, updated_at, email, password_hash;

-- name: GetUserByEmail :one
select id, created_at, updated_at, email, password_hash
from v1.users
where lower(email) = lower(sqlc.arg(email));

-- name: GetUserByID :one
select id, created_at, updated_at, email, password_hash
from v1.users
where id = sqlc.arg(id);

-- name: CreateUserLoginLog :exec
insert into v1.user_login_logs (user_id)
values (sqlc.arg(user_id));
