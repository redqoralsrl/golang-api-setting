-- name: CreateUser :one
insert into v1.users (
    email,
    password_hash
) values (
    $1,
    $2
)
returning
    id,
    created_at,
    updated_at,
    email,
    password_hash;

-- name: CreateUserLoginLog :exec
insert into v1.user_login_logs (
    user_id
)
values ($1);