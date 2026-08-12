import { NextResponse, type NextRequest } from "next/server";

const PLATFORM_HOST_PREFIX = "app.";

function parseHost(hostHeader: string): string {
  return hostHeader.split(":")[0]?.toLowerCase() ?? "";
}

/**
 * Extract tenant slug from Host.
 * - app.localhost / app.{BASE_DOMAIN} → platform (no tenant)
 * - {slug}.localhost / {slug}.{BASE_DOMAIN} → tenant slug
 */
export function resolveTenantSlug(
  host: string,
  baseDomain: string,
): string | null {
  const h = host.toLowerCase();
  const base = baseDomain.toLowerCase();

  if (h === `${PLATFORM_HOST_PREFIX}${base}` || h === "app.localhost") {
    return null;
  }

  // *.localhost
  if (h.endsWith(".localhost")) {
    const slug = h.slice(0, -".localhost".length);
    if (!slug || slug.includes(".")) return null;
    return slug;
  }

  // *.{BASE_DOMAIN}
  const suffix = `.${base}`;
  if (base && h.endsWith(suffix)) {
    const slug = h.slice(0, -suffix.length);
    if (!slug || slug.includes(".")) return null;
    if (slug === "app") return null;
    return slug;
  }

  return null;
}

export function middleware(request: NextRequest) {
  const host = parseHost(request.headers.get("host") ?? "");
  const baseDomain =
    process.env.NEXT_PUBLIC_BASE_DOMAIN ??
    process.env.APP_BASE_DOMAIN ??
    "localhost";

  const slug = resolveTenantSlug(host, baseDomain);
  const requestHeaders = new Headers(request.headers);

  if (slug) {
    requestHeaders.set("x-tenant-slug", slug);
  } else {
    requestHeaders.delete("x-tenant-slug");
  }

  // Platform host marker for layouts that need it
  const isPlatform =
    host === `app.${baseDomain}` || host === "app.localhost";
  if (isPlatform) {
    requestHeaders.set("x-platform-host", "1");
  }

  return NextResponse.next({
    request: { headers: requestHeaders },
  });
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon.ico|icons/|manifest.json|sw.js|workbox-.*|swe-worker-.*).*)",
  ],
};
