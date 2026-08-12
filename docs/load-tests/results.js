import http from "k6/http";
import { check, sleep } from "k6";

/**
 * Baseline /results load test.
 * Env: BASE_URL, TOKEN, TENANT_HOST
 */
export const options = {
  scenarios: {
    results_burst: {
      executor: "constant-vus",
      vus: Number(__ENV.VUS || 200),
      duration: __ENV.DURATION || "60s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<500"],
  },
};

const BASE = __ENV.BASE_URL || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "";
const HOST = __ENV.TENANT_HOST || "bitmesra.localhost";

export default function () {
  const res = http.get(`${BASE}/api/v1/results/me`, {
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      Host: HOST,
    },
  });
  check(res, {
    "status 200": (r) => r.status === 200,
  });
  sleep(0.05);
}
