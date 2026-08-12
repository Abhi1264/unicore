# Unicore Web

Next.js App Router frontend for the Unicore multi-tenant college ERP.

## Docs

Project documentation lives at the monorepo root:

- [Architecture](../../docs/architecture.md)
- [API](../../docs/api.md)
- [Local development](../../docs/local-development.md)
- [Contributing](../../CONTRIBUTING.md)
- [Deploy](../../infra/DEPLOY.md)

## Develop

From the repo root:

```bash
pnpm install
pnpm --filter web dev
```

Open `http://app.localhost:3000` (platform) or `http://demo.localhost:3000` (tenant).

Environment:

| Variable | Purpose |
| --- | --- |
| `NEXT_PUBLIC_API_URL` | Go API base (default `http://localhost:8080`) |
| `NEXT_PUBLIC_BASE_DOMAIN` | Host suffix for tenant slug parsing (default `localhost`) |

## Build

```bash
pnpm --filter web build
```

Produces a standalone Next.js output (`output: "standalone"`) with PWA assets via `@ducanh2912/next-pwa`. The build script uses `next build --webpack` because the PWA plugin injects webpack configuration.