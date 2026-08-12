# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS builder
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget \
 && useradd -u 1000 -m unicore \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /out/worker /app/worker
COPY apps/api/internal/db/migrations /app/migrations
USER unicore
EXPOSE 8080
# CMD rather than ENTRYPOINT: the same image runs both binaries, and the worker
# service selects its own via `command:`. An ENTRYPOINT would turn that into an
# argument to the server and silently start a second API instead of the worker.
CMD ["/app/server"]
