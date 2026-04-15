-- name: CreateErrorLog :one
insert into v1.error_log (
    timestamp,
    ip_address,
    user_agent,
    path,
    http_method,
    requested_url,
    error_code,
    error_message
) values (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
returning
    id,
    timestamp,
    ip_address,
    user_agent,
    path,
    http_method,
    requested_url,
    error_code,
    error_message;
