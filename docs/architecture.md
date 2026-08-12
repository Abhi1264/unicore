# Unicore Architecture

Unicore is a multi-tenant campus platform (results, registration, institute admin) built as a monorepo:

| Layer | Stack |
| --- | --- |
| API / worker | Go (Fiber), sqlc, pgx |
| Web | Next.js |
| Data | Postgres 16 (shared DB + RLS) |
| Cache | Redis 7 (cache-aside) |
| Events | NATS JetStream + transactional outbox |
| Shared contracts | `@unicore/shared` (Zod) |

This document is the interview-ready mental model for how tenancy, auth, cache, and async work fit together.

---

## Multi-tenancy: shared database + RLS

**Model:** one Postgres database, one schema, every tenant-owned row carries `tenant_id`. Isolation is enforced by **Row Level Security (RLS)**, not by separate databases.

**Why this shape:**

- Operationally simple: one pool, one migration path, one backup story.
- Cheap at early scale; tenants share connection capacity.
- Isolation is a database invariant, not an application convention that can be forgotten in one query.

**Application DB role:** the API connects as `unicore_app`, which must **not** have `BYPASSRLS`. Superuser/migration roles may bypass RLS for schema work; runtime traffic must not.

**Typical policy pattern:**

```sql
CREATE POLICY tenant_isolation ON enrollments
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
```

Every request that touches tenant data must set the session GUC before queries run (see below). If `app.tenant_id` is unset, policies fail closed (no rows).

---

## Request path: host → JWT → SET LOCAL

```
Browser
  └─ Host: mit.localhost (or mit.<base-domain>)
       └─ API middleware
            1. Resolve tenant from Host / PLATFORM_HOST
            2. Authenticate JWT (if present)
            3. Require JWT.tenant_id == resolved tenant (except platform superadmin flows)
            4. BEGIN; SET LOCAL app.tenant_id = '<uuid>';
            5. Handler → service → sqlc / cache / outbox
            6. COMMIT / ROLLBACK
```

### Host-based tenant resolution

- Tenant traffic: `{subdomain}.{APP_BASE_DOMAIN}` (locally `mit.localhost`).
- Platform host: `PLATFORM_HOST` (e.g. `app.localhost`) for superadmin / self-serve registration.
- Resolution looks up `tenants.subdomain` (cached in Redis when healthy).

### JWT tenant match

Access tokens embed at least: `sub` (user id), `tenant_id`, `role`, expiry. After host resolution, middleware compares claims:

- Mismatch → `401` / `403` with a stable error code (never leak another tenant’s existence beyond “unauthorized”).
- `superadmin` on the platform host may operate cross-tenant via explicit APIs; those paths still set `app.tenant_id` to the *target* tenant for the duration of the transaction.

### `SET LOCAL app.tenant_id`

Use **`SET LOCAL`** inside a transaction so the GUC is scoped to that transaction and cleared on commit/rollback. Do not rely on a long-lived connection setting alone—pooled connections would leak tenant context across requests.

```sql
BEGIN;
SET LOCAL app.tenant_id = '11111111-1111-1111-1111-111111111111';
-- all sqlc queries in this tx see only this tenant via RLS
COMMIT;
```

**Invariant:** handlers never pass `tenant_id` as a “filter you hope someone remembers.” Services receive a request context that already established the GUC; queries are written without `WHERE tenant_id = $x` as the sole isolation mechanism (RLS is the backstop; explicit filters remain fine for clarity/indexes).

---

## Redis: cache-aside

Pattern for hot reads (tenant config, branding, published results summaries):

1. Build a namespaced key: `tenant:{id}:results:{student_id}:{sem}` (always include tenant).
2. `GET` → hit → return.
3. Miss → query Postgres (under RLS) → `SET` with TTL → return.
4. On write / publish / admin mutation → **invalidate** (delete key or bump version prefix).

**Rules:**

- Cache is an optimization, never the source of truth.
- Keys are tenant-scoped; a bug that omits tenant from the key is a security defect.
- Treat Redis as optional (see graceful degradation).

---

## NATS + transactional outbox

Side effects (notifications, PDF generation, search index updates, webhooks) must not be “publish then hope the DB commit succeeds.”

**Outbox flow:**

1. In the same DB transaction as the business write, insert an `outbox` row (`id`, `tenant_id`, `topic`, `payload`, `created_at`, `published_at` NULL).
2. Commit.
3. Worker (or API sidecar loop) polls unpublished rows, publishes to NATS JetStream, marks `published_at`.
4. Consumers are idempotent (dedupe on event id / Idempotency-Key where applicable).

**Why outbox:** at-least-once delivery with a single source of truth. If NATS is down, rows accumulate and drain when the broker recovers—no dual-write drift.

---

## Graceful degradation

| Dependency | If unhealthy | Behavior |
| --- | --- | --- |
| Redis | down / timeout | Skip cache; read/write Postgres directly; log + metric |
| NATS | down | Accept writes; outbox retains events; delay async side effects |
| Postgres | down | Fail health checks; return `503` with `request_id` |

The API should remain useful for critical sync paths (login, results read) when Redis/NATS flap. Never fail a read solely because cache is unavailable. Never drop a committed business event because publish failed—leave it in the outbox.

---

## Layering (API)

```
handlers/     HTTP parse, authz gates, status codes
services/     use-cases, transactions, cache + outbox orchestration
db/           sqlc queries, migrations
cache/        Redis adapters
queue/        NATS publish/subscribe + outbox poller
middleware/   tenant resolve, JWT, request_id, rate limit
```

Handlers stay thin. Isolation, idempotency, and tenancy rules live in middleware + services + DB policies—not in ad-hoc handler SQL.

---

## Observability

- Every response can carry `request_id` (also on `ApiError`).
- Prometheus scrapes `api:8080/metrics` (see `infra/observability`).
- Prefer RED metrics (rate, errors, duration) plus tenant-resolution and cache hit-ratio counters.

---

## Security checklist (tenancy)

1. Runtime DB role cannot bypass RLS.
2. Every tenant request sets `SET LOCAL app.tenant_id`.
3. JWT `tenant_id` matches Host-resolved tenant.
4. Cache keys always include tenant.
5. Outbox/events carry `tenant_id`; consumers re-enter with the correct GUC.
6. Integration tests cover cross-tenant denial (isolation) and duplicate `Idempotency-Key` (idempotency).
