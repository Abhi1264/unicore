# Unicore API

Go + Fiber API and NATS worker for Unicore.

See root [README](../../README.md) and [docs](../../docs/).

```bash
export DATABASE_URL=postgres://unicore:unicore@localhost:5432/unicore?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export NATS_URL=nats://localhost:4222
go run ./cmd/server
go run ./cmd/worker
go run ./cmd/seed
```
