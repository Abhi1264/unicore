# Deploy

## Prerequisites

- Managed Postgres with daily backups and a tested restore path (see [Backups](#backups--restore))
- Redis and NATS reachable only from the API/worker network
- Distinct secrets for JWT access, JWT refresh, payment webhooks, and metrics scrape

## API + worker (Fly.io)

Config: [`fly.toml`](./fly.toml). Image: `apps/api/Dockerfile` (builds `server` and `worker`).

```bash
fly auth login
fly apps create unicore-api   # once

fly secrets set \
  APP_ENV=production \
  DATABASE_URL="postgres://unicore_app:…@…/unicore?sslmode=require" \
  DATABASE_MIGRATE_URL="postgres://unicore:…@…/unicore?sslmode=require" \
  REDIS_URL="redis://…" \
  NATS_URL="nats://…" \
  JWT_ACCESS_SECRET="…" \
  JWT_REFRESH_SECRET="…" \
  PAYMENT_WEBHOOK_SECRET="…" \
  METRICS_TOKEN="…" \
  WEB_URL="https://app.unicore.app" \
  APP_BASE_DOMAIN="unicore.app" \
  PLATFORM_HOST="app.unicore.app"

fly deploy --config infra/fly.toml --dockerfile apps/api/Dockerfile
```

Runtime notes:

- Health: `GET /healthz` on **8080**
- Metrics: `GET /metrics` with `Authorization: Bearer $METRICS_TOKEN`
- Worker: same image, command `/app/worker`
- DNS: `*.unicore.app` and `app.unicore.app` → Fly

### Migrations

Use the **owner** URL (`DATABASE_MIGRATE_URL`), never the RLS-bound app role:

```bash
# release / one-off machine
DATABASE_URL="$DATABASE_MIGRATE_URL" /app/server   # boots migrate-up then exits if you use a migrate-only entry
# or:
migrate -path /app/migrations -database "$DATABASE_MIGRATE_URL" up
```

The API process also migrates on boot when `DATABASE_MIGRATE_URL` is set. Prefer an explicit release step in production so a bad migration fails before traffic shifts.

Rollback:

1. Take a snapshot / confirm PITR window before migrating.
2. Prefer expand/contract schema changes (add column → dual-write → drop old).
3. For a reversible migration: `migrate down 1` against `DATABASE_MIGRATE_URL`, then redeploy the previous image.
4. If data was rewritten in place, restore from backup — `down` SQL alone is not enough.

## Web (Vercel)

Root directory: `apps/web`.

```bash
cd apps/web
vercel link
# Production env:
#   NEXT_PUBLIC_API_URL=https://api.unicore.app
#   NEXT_PUBLIC_BASE_DOMAIN=unicore.app
vercel --prod
```

Only `NEXT_PUBLIC_*` values are browser-visible. Never put JWT or DB secrets there.

## Backups & restore

| Asset | Minimum |
|-------|---------|
| Postgres | Daily automated backups + PITR; restore drill quarterly |
| Object/file storage (`STORAGE_PATH`) | Volume snapshots or object-store versioning |
| Redis | Ephemeral OK (sessions/rate limits); no durable source of truth |

Document RPO/RTO with your host. After restore: run `migrate up` to current schema, invalidate refresh tokens if credentials may have leaked, and verify one tenant login end-to-end.

## Observability

Local: `docker compose --profile observability up -d`  
Prod: scrape `/metrics` with the bearer token; correlate client errors via `request_id`.
