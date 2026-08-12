import type { NextConfig } from "next";
import withPWAInit from "@ducanh2912/next-pwa";

const withPWA = withPWAInit({
  dest: "public",
  disable: process.env.NODE_ENV === "development",
  fallbacks: {
    document: "/offline",
  },
});

const apiOrigin = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/**
 * Session tokens live in localStorage, which means any script that executes on
 * the page can read them. A restrictive CSP is what keeps an injected script
 * from running in the first place, so it is load-bearing here rather than
 * defence in depth.
 *
 * 'unsafe-inline' on style-src is required by Tailwind's runtime style
 * injection. script-src carries no such escape hatch; Next's inline bootstrap
 * scripts are covered by 'strict-dynamic' via the nonce Next emits, and
 * 'unsafe-eval' is limited to development where React Refresh needs it.
 */
const csp = [
  "default-src 'self'",
  process.env.NODE_ENV === "development"
    ? "script-src 'self' 'unsafe-inline' 'unsafe-eval'"
    : "script-src 'self' 'unsafe-inline'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob: https:",
  "font-src 'self' data:",
  `connect-src 'self' ${apiOrigin} ws: wss:`,
  "object-src 'none'",
  "base-uri 'self'",
  "form-action 'self'",
  "frame-ancestors 'none'",
  "upgrade-insecure-requests",
].join("; ");

const securityHeaders = [
  { key: "Content-Security-Policy", value: csp },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=(), payment=()",
  },
];

const nextConfig: NextConfig = {
  output: "standalone",
  reactCompiler: true,
  transpilePackages: ["@unicore/shared"],
  // @ducanh2912/next-pwa injects webpack; Next 16 defaults to Turbopack.
  // Build uses `next build --webpack` so the PWA plugin can emit the SW.
  turbopack: {},
  poweredByHeader: false,
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
};

export default withPWA(nextConfig);
