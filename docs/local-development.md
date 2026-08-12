# Local Development

## Prerequisites

- Docker + Docker Compose
- Go 1.26+ (API / worker)
- Node.js 20+ and pnpm (`packageManager` in root `package.json`)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI
- [sqlc](https://sqlc.dev/) CLI
- Optional: k6 for load tests

Copy env defaults:

```bash
cp .env.example .env
```

---

## 1. Start infrastructure

From the repo root:

```bash
# Postgres, Redis, NATS, API, worker, web
docker compose up --build

# Or infra only (run API/web on the host)
docker compose up -d postgres redis nats
```

With the observability profile (Prometheus + Grafana):

```bash
docker compose --profile observability up -d
```

| Service | Host port |
| --- | --- |
| Postgres | `5432` |
| Redis | `6379` |
| NATS client | `4222` |
| NATS monitor | `8222` |
| API | `8080` |
| Web | `3000` |
| Prometheus | `9090` (profile) |
| Grafana | `3001` (profile; set `GRAFANA_ADMIN_PASSWORD`) |

Health: `curl -s http://localhost:8080/healthz`

---

## 2. Hostnames for multi-tenant routing

Unicore resolves tenants from the `Host` header (`{subdomain}.{APP_BASE_DOMAIN}`). Locally `APP_BASE_DOMAIN=localhost` and `PLATFORM_HOST=app.localhost`.

Modern browsers resolve `*.localhost` to `127.0.0.1` without `/etc/hosts`. Prefer:

```text
http://app.localhost:3000          # platform / registration
http://bitmesra.localhost:3000     # demo tenant (after seeding)
http://app.localhost:8080          # API on platform host
http://bitmesra.localhost:8080     # API on tenant host
```

If your OS/browser does **not** resolve subdomain.localhost, add entries:

```bash
# /etc/hosts
127.0.0.1 app.localhost
127.0.0.1 demo.localhost
127.0.0.1 mit.localhost
```

Point API clients at the correct Host (curl example):

```bash
curl -s -H "Host: demo.localhost" http://127.0.0.1:8080/healthz
```

---

## 3. Migrations

Migrations live in `apps/api/internal/db/migrations` (golang-migrate numbered files).

Against local Compose Postgres:

```bash
export DATABASE_URL="postgres://unicore:unicore@localhost:5432/unicore?sslmode=disable"

migrate -path apps/api/internal/db/migrations \
  -database "$DATABASE_URL" up
```

Down one step:

```bash
migrate -path apps/api/internal/db/migrations \
  -database "$DATABASE_URL" down 1
```

`infra/postgres/init.sh` creates the non-bypass RLS role `unicore_app` on first volume init. Recreate volumes if you need a clean DB:

```bash
docker compose down -v
docker compose up -d postgres redis nats
```

### Demo data

```bash
cd apps/api
DATABASE_MIGRATE_URL='postgres://unicore:unicore@127.0.0.1:5432/unicore?sslmode=disable' \
  SEED_STUDENTS=500 go run ./cmd/seed
```

Idempotent enough to re-run. Passwords are `Unicore-<role>-2026!` (see root README).

---

## 4. sqlc

Config: `apps/api/sqlc.yaml`. Queries: `apps/api/internal/db/queries`. Generated package: `apps/api/internal/db/sqlcdb`.

```bash
cd apps/api
sqlc generate
```

Regenerate after changing SQL queries or schema; commit generated Go when the team’s convention requires it.

---

## 5. Run API, worker, and web on the host

Keep Compose dependencies up (`postgres`, `redis`, `nats`). Use `.env` / `.env.example` values with `localhost` hosts (not Docker service names).

### API

```bash
cd apps/api
# ensure migrations + sqlc are current
go run ./cmd/server
```

Listens on `API_ADDR` (default `:8080`).

### Worker

```bash
cd apps/api
go run ./cmd/worker
```

Consumes outbox → NATS and background jobs. Same `DATABASE_URL` / `REDIS_URL` / `NATS_URL` as the API.

### Web

```bash
# from repo root
pnpm install
pnpm --filter web dev
# or: pnpm dev:web
```

Defaults: `NEXT_PUBLIC_API_URL=http://localhost:8080`, `NEXT_PUBLIC_BASE_DOMAIN=localhost`. Open `http://app.localhost:3000` (or `http://localhost:3000` if you are not testing host routing yet).

### Shared package

```bash
pnpm --filter @unicore/shared typecheck
```

Wire `@unicore/shared` from the web app via the workspace (`packages/*` in `pnpm-workspace.yaml`).

---

## 6. Useful checks

```bash
# Infra
docker compose ps
redis-cli ping
curl -s http://localhost:8222/healthz

# API metrics (when instrumentation is enabled)
curl -s http://localhost:8080/metrics | head
```

Load tests: see [docs/load-tests/README.md](./load-tests/README.md).

Architecture deep dive: [docs/architecture.md](./architecture.md).
