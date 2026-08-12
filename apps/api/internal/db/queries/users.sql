-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, role, full_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, email, password_hash, role, full_name, is_active, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, tenant_id, email, password_hash, role, full_name, is_active, created_at, updated_at
FROM users
WHERE tenant_id = $1 AND email = $2;

-- name: GetUserByID :one
SELECT id, tenant_id, email, password_hash, role, full_name, is_active, created_at, updated_at
FROM users
WHERE tenant_id = $1 AND id = $2;

-- name: InsertLoginEvent :exec
INSERT INTO login_events (tenant_id, user_id, ip, user_agent)
VALUES ($1, $2, $3, $4);
