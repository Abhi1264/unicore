import http from "k6/http";
import { check } from "k6";
import { uuidv4 } from "https://jslib.k6.io/k6-utils/1.4.0/index.js";

/**
 * Concurrent course registration against a small seat cap.
 * Env: BASE_URL, TOKEN, TENANT_HOST, COURSE_ID, SEMESTER
 */
export const options = {
  scenarios: {
    enroll_race: {
      executor: "per-vu-iterations",
      vus: Number(__ENV.VUS || 100),
      iterations: 1,
      maxDuration: "2m",
    },
  },
};

const BASE = __ENV.BASE_URL || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "";
const HOST = __ENV.TENANT_HOST || "bitmesra.localhost";
const COURSE_ID = __ENV.COURSE_ID || "";
const SEMESTER = __ENV.SEMESTER || "2026S1";

export default function () {
  const res = http.post(
    `${BASE}/api/v1/enrollments`,
    JSON.stringify({ course_id: COURSE_ID, semester: SEMESTER }),
    {
      headers: {
        Authorization: `Bearer ${TOKEN}`,
        Host: HOST,
        "Content-Type": "application/json",
        "Idempotency-Key": uuidv4(),
      },
    }
  );
  check(res, {
    "accepted or full": (r) => r.status === 200 || r.status === 201 || r.status === 409,
  });
}
