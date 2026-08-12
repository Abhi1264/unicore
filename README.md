# Unicore

Open-source multi-tenant college ERP — built for real concurrent load (result day, fee deadlines, registration windows), clean UX, and true tenant isolation via Postgres RLS.

## Quick start

```bash
cp .env.example .env
docker compose up --build

# Seed two demo campuses (depts, courses, students, enrollments, results,
# attendance, fees, announcements). Defaults: 200 students / tenant.
cd apps/api
DATABASE_MIGRATE_URL='postgres://unicore:unicore@127.0.0.1:5432/unicore?sslmode=disable' \
  SEED_STUDENTS=500 go run ./cmd/seed
```

Scale knobs: `SEED_STUDENTS`, `SEED_FACULTY`, `SEED_ATTENDANCE_STUDENTS`, `SEED_ATTENDANCE_SESSIONS`, `SEED_PASSWORD_PREFIX`.

- Web: http://localhost:3000
- API: http://localhost:8080/healthz
- Metrics: http://localhost:8080/metrics

Demo hosts (`*.localhost` resolves in modern browsers):

| Host | Purpose |
|------|---------|
| http://bitmesra.localhost:3000 | Demo tenant 1 |
| http://demo2.localhost:3000 | Demo tenant 2 |
| http://app.localhost:3000 | Platform / signup / superadmin |

Seeded logins (per tenant slug):

| Role | Email | Password |
|------|-------|----------|
| Institute admin | `admin@{slug}.edu` | `Unicore-admin-2026!` |
| Faculty | `faculty@{slug}.edu` | `Unicore-faculty-2026!` |
| Student | `student@{slug}.edu` | `Unicore-student-2026!` |
| Bulk students | `s0000@{slug}.edu` … | `Unicore-student-2026!` |
| Superadmin | `superadmin@unicore.local` on `app.localhost` | `Unicore-superadmin-2026!` |

Observability stack (optional):

```bash
docker compose --profile observability up -d
```

## Architecture

Shared database, shared schema, **row-level security** keyed by `app.tenant_id`. Subdomain (or custom domain) resolves the tenant; JWTs carry `tenant_id` and must match the Host. Hot reads use Redis cache-aside; payments, PDFs, bulk CSV, and notification fan-out go through NATS + a transactional outbox.

See [docs/architecture.md](docs/architecture.md).

```
Browser (Next.js PWA)
   → API (Go/Fiber) → Postgres (RLS) / Redis / NATS
                         ↑
                      Worker
```

## Apps

| Path | Role |
|------|------|
| `apps/api` | Go API + worker |
| `apps/web` | Next.js App Router frontend |
| `packages/shared` | Shared Zod/TS contracts |

## Docs

- [Local development](docs/local-development.md)
- [API](docs/api.md)
- [Load tests](docs/load-tests/README.md)
- [Contributing](CONTRIBUTING.md)
- [Deploy](infra/DEPLOY.md)

## Load testing

k6 scripts live under `docs/load-tests/`. Capture uncached vs cached `/results` and seat-cap registration races; check numbers into `docs/load-tests/reports/`.

## License

MIT
