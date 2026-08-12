import { ApiErrorSchema, type ApiError } from "@unicore/shared";
import { getAccessToken, getRefreshToken, setTokens, clearSession } from "./auth";

export class ApiRequestError extends Error {
  status: number;
  code?: string;
  requestId?: string;

  constructor(status: number, body: ApiError) {
    super(body.error);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = body.code;
    this.requestId = body.request_id;
  }
}

function apiBaseUrl(): string {
  const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  return base.replace(/\/$/, "");
}

/** Tenant slug from the browser host. */
export function tenantSlugFromLocation(): string | null {
  if (typeof window === "undefined") return null;
  const host = window.location.hostname.toLowerCase();
  if (host === "localhost" || host === "127.0.0.1") return null;
  if (host.endsWith(".localhost")) {
    const slug = host.slice(0, -".localhost".length);
    if (!slug || slug.includes(".") || slug === "app") return null;
    return slug;
  }
  const base = (
    process.env.NEXT_PUBLIC_BASE_DOMAIN ?? "localhost"
  ).toLowerCase();
  const suffix = `.${base}`;
  if (base && host.endsWith(suffix)) {
    const slug = host.slice(0, -suffix.length);
    if (!slug || slug.includes(".") || slug === "app") return null;
    return slug;
  }
  return null;
}

/** Origin for platform pages (derived from the current base domain). */
export function platformOrigin(): string {
  if (typeof window === "undefined") return "/";
  const { protocol, port } = window.location;
  const base = (process.env.NEXT_PUBLIC_BASE_DOMAIN ?? "localhost").toLowerCase();
  return `${protocol}//app.${base}${port ? `:${port}` : ""}`;
}

export type ApiFetchOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
  auth?: boolean;
  idempotencyKey?: string;
};

async function parseError(res: Response): Promise<ApiRequestError> {
  let body: ApiError = { error: res.statusText || "Request failed" };
  try {
    const json: unknown = await res.json();
    const parsed = ApiErrorSchema.safeParse(json);
    if (parsed.success) body = parsed.data;
    else if (json && typeof json === "object" && "error" in json) {
      body = {
        error: String((json as { error: unknown }).error),
        code:
          "code" in json ? String((json as { code: unknown }).code) : undefined,
        request_id:
          "request_id" in json
            ? String((json as { request_id: unknown }).request_id)
            : undefined,
      };
    }
  } catch {
    /* keep default */
  }
  return new ApiRequestError(res.status, body);
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  const refresh = getRefreshToken();
  if (!refresh) return false;
  try {
    const res = await fetch(`${apiBaseUrl()}/api/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (!res.ok) return false;
    const tokens = (await res.json()) as {
      access_token: string;
      refresh_token: string;
      token_type?: string;
      expires_in: number;
    };
    setTokens({
      access_token: tokens.access_token,
      refresh_token: tokens.refresh_token,
      token_type: "Bearer",
      expires_in: tokens.expires_in,
    });
    return true;
  } catch {
    return false;
  }
}

export async function apiFetch<T = unknown>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const { body, auth = true, idempotencyKey, headers: initHeaders, ...rest } =
    options;

  const headers = new Headers(initHeaders);
  // Do not set Content-Type on FormData — the browser must add the multipart boundary.
  const isFormData = typeof FormData !== "undefined" && body instanceof FormData;
  if (body !== undefined && !isFormData && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (idempotencyKey) {
    headers.set("Idempotency-Key", idempotencyKey);
  }
  if (!headers.has("X-Tenant-Slug")) {
    const slug = tenantSlugFromLocation();
    if (slug) headers.set("X-Tenant-Slug", slug);
  }
  if (auth) {
    const token = getAccessToken();
    if (token) headers.set("Authorization", `Bearer ${token}`);
  }

  const url = path.startsWith("http") ? path : `${apiBaseUrl()}${path}`;

  const doFetch = () =>
    fetch(url, {
      ...rest,
      credentials: "include",
      headers,
      body:
        body === undefined
          ? undefined
          : isFormData
            ? (body as FormData)
            : JSON.stringify(body),
    });

  let res = await doFetch();

  if (res.status === 401 && auth) {
    if (!refreshPromise) {
      refreshPromise = tryRefresh().finally(() => {
        refreshPromise = null;
      });
    }
    const ok = await refreshPromise;
    if (ok) {
      const token = getAccessToken();
      if (token) headers.set("Authorization", `Bearer ${token}`);
      res = await doFetch();
    } else {
      clearSession();
    }
  }

  if (!res.ok) {
    throw await parseError(res);
  }

  if (res.status === 204) return undefined as T;
  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

/** Downloads a protected file with the bearer token (nav alone would 401). */
export async function downloadFile(
  path: string,
  filename: string,
): Promise<void> {
  const headers = new Headers();
  const slug = tenantSlugFromLocation();
  if (slug) headers.set("X-Tenant-Slug", slug);
  const token = getAccessToken();
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${apiBaseUrl()}${path}`, {
    credentials: "include",
    headers,
  });
  if (!res.ok) throw await parseError(res);

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  try {
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    // Delayed revoke: immediate revoke can cancel the download in some browsers.
    setTimeout(() => URL.revokeObjectURL(url), 10_000);
  }
}
