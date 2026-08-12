-- name: GetTenantBySlug :one
SELECT id, slug, name, custom_domain, status, plan_tier, created_at, updated_at
FROM tenants
WHERE slug = $1;

-- name: GetTenantByCustomDomain :one
SELECT id, slug, name, custom_domain, status, plan_tier, created_at, updated_at
FROM tenants
WHERE custom_domain = $1;

-- name: GetTenantByID :one
SELECT id, slug, name, custom_domain, status, plan_tier, created_at, updated_at
FROM tenants
WHERE id = $1;

-- name: CreateTenant :one
INSERT INTO tenants (slug, name, status)
VALUES ($1, $2, 'pending_approval')
RETURNING id, slug, name, custom_domain, status, plan_tier, created_at, updated_at;

-- name: UpdateTenantStatus :one
UPDATE tenants SET status = $2, updated_at = now()
WHERE id = $1
RETURNING id, slug, name, custom_domain, status, plan_tier, created_at, updated_at;

-- name: ListTenants :many
SELECT id, slug, name, custom_domain, status, plan_tier, created_at, updated_at
FROM tenants
ORDER BY created_at DESC;

-- name: ListPendingTenants :many
SELECT id, slug, name, custom_domain, status, plan_tier, created_at, updated_at
FROM tenants
WHERE status = 'pending_approval'
ORDER BY created_at ASC;

-- name: CreateTenantConfig :one
INSERT INTO tenant_config (tenant_id, grading_system, academic_calendar_type, branding, grading_scale, attendance_threshold_pct)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetTenantConfig :one
SELECT * FROM tenant_config WHERE tenant_id = $1;

-- name: UpdateTenantConfig :one
UPDATE tenant_config
SET grading_system = COALESCE(sqlc.narg(grading_system), grading_system),
    academic_calendar_type = COALESCE(sqlc.narg(academic_calendar_type), academic_calendar_type),
    branding = COALESCE(sqlc.narg(branding), branding),
    grading_scale = COALESCE(sqlc.narg(grading_scale), grading_scale),
    attendance_threshold_pct = COALESCE(sqlc.narg(attendance_threshold_pct), attendance_threshold_pct),
    updated_at = now()
WHERE tenant_id = $1
RETURNING *;
