# Unicore API

Base URL (local): `http://localhost:8080`  
Prefix: `/api/v1`  
Contracts: `@unicore/shared`

## Host-based tenant resolution

| Host | Meaning |
| --- | --- |
| `{slug}.{APP_BASE_DOMAIN}` | Tenant context (e.g. `bitmesra.localhost`) |
| Custom domain | Resolved via `tenants.custom_domain` |
| `PLATFORM_HOST` | Platform (e.g. `app.localhost`) — signup, superadmin |

Authenticated tenant routes require JWT `tenant_id` to match the Host-resolved tenant (superadmin exempt on platform host).

Optional header on platform host: `X-Tenant-Slug` for tooling.

## Error shape

```json
{ "error": "human message", "code": "HTTP_ERROR", "request_id": "…" }
```

## Idempotency

Send `Idempotency-Key` on:

- `POST /api/v1/fees/pay`
- `POST /api/v1/enrollments`

Duplicates return the original resource.

## Auth

| Method | Path | Notes |
| --- | --- | --- |
| POST | `/api/v1/auth/register-tenant` | Platform host; creates `pending_approval` tenant + admin |
| POST | `/api/v1/auth/login` | Tenant host |
| POST | `/api/v1/auth/refresh` | Rotate refresh |
| GET | `/api/v1/auth/me` | JWT |

## Tenants / superadmin

| Method | Path | Role |
| --- | --- | --- |
| GET | `/api/v1/tenants/current` | JWT |
| PATCH | `/api/v1/tenants/current/branding` | institute_admin |
| GET | `/api/v1/admin/tenants/` | superadmin |
| POST | `/api/v1/admin/tenants/:id/approve\|reject\|suspend\|reactivate` | superadmin |
| GET | `/api/v1/admin/usage` | superadmin |
| GET | `/api/v1/admin/audit-logs` | institute_admin, superadmin |
| POST | `/api/v1/admin/bulk-import` | institute_admin |

## Academic / results / fees / etc.

| Area | Paths |
| --- | --- |
| Results | `GET /results/me`, `POST /results`, `POST /results/publish` |
| Departments/courses | `GET\|POST /departments`, `GET\|POST /courses`, `GET /courses/:id/roster` |
| Enrollment | `POST /enrollments`, `POST /enrollments/drop` |
| Timetable | `GET\|POST /timetable` |
| Registration windows | `GET\|POST /registration-windows`, `GET /registration-windows/open` |
| Fees | `GET\|POST /fees/heads`, `GET /fees/dues`, `POST /fees/pay`, `POST /fees/confirm` |
| Attendance | `POST /attendance`, `GET /attendance/summary` |
| Announcements | `GET\|POST /announcements`, `GET /announcements/stream` (SSE) |
| Documents | `GET\|POST /documents`, `GET /documents/:id/download` |

## Ops

- `GET /healthz` — liveness
- `GET /readyz` — DB readiness
- `GET /metrics` — Prometheus
