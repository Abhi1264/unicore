# Load tests (k6)

Scripts in this directory stress the Unicore API under multi-tenant Host routing. Focus areas:

1. **Published results reads** — cache-aside + RLS path (`GET /v1/results/...`)
2. **Tenant registration** — platform host + `Idempotency-Key` (`POST /v1/auth/register-tenant`)

## Prerequisites

- [k6](https://k6.io/docs/get-started/installation/) installed locally
- API reachable (Compose `api` or `go run ./cmd/server`)
- Seeded tenant for results tests (example: subdomain `demo`, student credentials in env)
- `/etc/hosts` or `*.localhost` resolution working (see [local-development.md](../local-development.md))

## Environment

| Variable | Default | Purpose |
| --- | --- | --- |
| `API_BASE` | `http://127.0.0.1:8080` | Dial address (IP/localhost is fine) |
| `TENANT_HOST` | `demo.localhost` | `Host` header for tenant routes |
| `PLATFORM_HOST` | `app.localhost` | `Host` header for registration |
| `STUDENT_EMAIL` | — | Login for results scenarios |
| `STUDENT_PASSWORD` | — | Login for results scenarios |
| `VUS` | `20` | Virtual users |
| `DURATION` | `1m` | Test duration |

k6 does not use the system resolver the same way browsers do for `*.localhost` in all setups. Prefer dialing `127.0.0.1:8080` and setting the `Host` header explicitly in scripts.

## Run: results

```bash
cd docs/load-tests

API_BASE=http://127.0.0.1:8080 \
TENANT_HOST=demo.localhost \
STUDENT_EMAIL=student@demo.edu \
STUDENT_PASSWORD='password123' \
k6 run results.js
```

Expected script behavior:

1. Setup: `POST /v1/auth/login` with `Host: $TENANT_HOST`.
2. Default function: `GET /v1/results/me` (or semester-scoped path) with `Authorization` + tenant `Host`.
3. Thresholds: p95 latency and error rate (tune when baselines exist).

If `results.js` is not checked in yet, start from:

```javascript
import http from "k6/http";
import { check, sleep } from "k6";

const API_BASE = __ENV.API_BASE || "http://127.0.0.1:8080";
const TENANT_HOST = __ENV.TENANT_HOST || "demo.localhost";

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || "1m",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
  },
};

export function setup() {
  const res = http.post(
    `${API_BASE}/v1/auth/login`,
    JSON.stringify({
      email: __ENV.STUDENT_EMAIL,
      password: __ENV.STUDENT_PASSWORD,
    }),
    {
      headers: {
        "Content-Type": "application/json",
        Host: TENANT_HOST,
      },
    },
  );
  check(res, { "login 200": (r) => r.status === 200 });
  const body = res.json();
  return { token: body.tokens.access_token };
}

export default function (data) {
  const res = http.get(`${API_BASE}/v1/results/me`, {
    headers: {
      Authorization: `Bearer ${data.token}`,
      Host: TENANT_HOST,
    },
  });
  check(res, { "results 200": (r) => r.status === 200 });
  sleep(0.3);
}
```

Save as `results.js` and run with the command above.

## Run: registration

Registration hits the **platform** host and should send a unique `Idempotency-Key` per logical signup. For pure load, generate a new subdomain + key per iteration; for idempotency verification, reuse one key and assert identical responses.

```bash
API_BASE=http://127.0.0.1:8080 \
PLATFORM_HOST=app.localhost \
k6 run registration.js
```

Sketch:

```javascript
import http from "k6/http";
import { check } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

const API_BASE = __ENV.API_BASE || "http://127.0.0.1:8080";
const PLATFORM_HOST = __ENV.PLATFORM_HOST || "app.localhost";

export const options = {
  vus: Number(__ENV.VUS || 5),
  duration: __ENV.DURATION || "30s",
  thresholds: {
    http_req_failed: ["rate<0.05"],
  },
};

export default function () {
  const id = uuidv4().slice(0, 8);
  const payload = JSON.stringify({
    institute_name: `Load Test ${id}`,
    subdomain: `lt${id}`,
    admin_email: `admin+${id}@example.com`,
    admin_full_name: "Load Admin",
    admin_password: "password123",
    timezone: "UTC",
  });

  const res = http.post(`${API_BASE}/v1/auth/register-tenant`, payload, {
    headers: {
      "Content-Type": "application/json",
      Host: PLATFORM_HOST,
      "Idempotency-Key": uuidv4(),
    },
  });

  check(res, {
    "register 201": (r) => r.status === 201 || r.status === 200,
  });
}
```

## Interpreting results

- Spike in latency with low error rate on results → check Redis hit ratio and Postgres plans.
- Errors with `503` / dependency codes → graceful degradation path; confirm API still serves cache misses from Postgres when Redis is stopped.
- Duplicate registrations under one `Idempotency-Key` → must not create two tenants.

Wire thresholds into CI only after you have a stable local baseline.
