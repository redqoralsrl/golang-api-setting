-- name: CreateErrorLog :exec
insert into v1.error_log (
    occurred_at,
    ip_address,
    user_agent,
    path,
    http_method,
    requested_url,
    error_code,
    error_message
) values (
    sqlc.arg(occurred_at),
    sqlc.arg(ip_address),
    sqlc.arg(user_agent),
    sqlc.arg(path),
    sqlc.arg(http_method),
    sqlc.arg(requested_url),
    sqlc.arg(error_code),
    sqlc.arg(error_message)
);
