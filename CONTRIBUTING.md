# Contributing to Unicore

Thanks for helping build Unicore. This guide covers structure, PR expectations, and the isolation/idempotency bar we hold ourselves to.

## Repository layout

```
apps/api/          Go API + worker
apps/web/          Next.js frontend
packages/shared/   Zod contracts (@unicore/shared)
docs/              Architecture, API, local dev, load tests
infra/             Postgres init, Fly, observability, deploy notes
```

## API layering

Keep dependencies flowing **inward** in one direction:

```
handlers → services → (db | cache | queue)
```

| Layer | Responsibility |
| --- | --- |
| `handlers/` | HTTP binding, validation against shared shapes, status codes. No business transactions. |
| `services/` | Use-cases, `BEGIN`/`SET LOCAL app.tenant_id`/`COMMIT`, orchestrate cache + outbox. |
| `db/` | Migrations, sqlc queries, RLS-aware access. |
| `cache/` | Redis get/set/invalidate; fail open to DB. |
| `queue/` | Outbox poller, NATS publish/subscribe. |
| `middleware/` | Tenant Host resolution, JWT, `request_id`, rate limits. |

Do not call Redis or NATS from handlers. Do not embed SQL in handlers. Do not bypass services “just this once.”

## Shared contracts

Request/response types that cross the wire belong in `packages/shared` (Zod). Prefer importing those schemas in the web app and documenting them in `docs/api.md` rather than inventing parallel TypeScript interfaces.

## Pull requests

1. **Small and focused** — one concern per PR when practical.
2. **Describe why** — summary + test plan (manual or automated).
3. **Match style** — follow neighboring code; no drive-by refactors unrelated to the change.
4. **Migrations** — include up/down; note any manual steps.
5. **Docs** — update `docs/` when behavior or ops steps change.
6. **Do not commit secrets** — use `.env.example` for new config keys only.

### Code quality

- No filler comments (`// increment counter`, narrating obvious code).
- Comment only non-obvious invariants (especially tenancy, idempotency, and degradation).
- Prefer clear names over clever abstractions.
- Errors: return `ApiError` shape `{ error, code?, request_id? }`.

## Tests (required for isolation & idempotency)

Any change that touches tenancy, authz, caching, or mutating endpoints **must** include tests that prove:

1. **Isolation** — tenant A credentials/Host cannot read or mutate tenant B data (RLS + JWT Host match). Prefer integration tests against real Postgres with RLS enabled.
2. **Idempotency** — replaying the same `Idempotency-Key` with the same body does not double-apply side effects (duplicate tenants, duplicate outbox events, duplicate enrollments).

Also cover:

- Cache miss path when Redis is unavailable (graceful degradation).
- Outbox row created in the same transaction as the business write.

Unit-only mocks of the DB are insufficient for isolation claims—RLS must be exercised.

## Local workflow

See [docs/local-development.md](./docs/local-development.md). Architecture primer: [docs/architecture.md](./docs/architecture.md).

```bash
docker compose up -d postgres redis nats
# migrate + sqlc generate
# run API, worker, web as documented
```

## License

By contributing, you agree that your contributions are licensed under the repository’s MIT license.
