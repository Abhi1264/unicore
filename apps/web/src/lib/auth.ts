import type { Role, TokenPair, UserPublic } from "@unicore/shared";

const ACCESS_KEY = "unicore_access_token";
const REFRESH_KEY = "unicore_refresh_token";
const USER_KEY = "unicore_user";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(ACCESS_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function setTokens(tokens: TokenPair): void {
  localStorage.setItem(ACCESS_KEY, tokens.access_token);
  localStorage.setItem(REFRESH_KEY, tokens.refresh_token);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
  localStorage.removeItem(USER_KEY);
  notifyUserChanged();
}

export function setUser(user: UserPublic): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  notifyUserChanged();
}

// Cache parsed user by raw string identity for useSyncExternalStore snapshots.
let cachedRaw: string | null = null;
let cachedUser: UserPublic | null = null;

export function getUser(): UserPublic | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(USER_KEY);
  if (raw === cachedRaw) return cachedUser;
  cachedRaw = raw;
  cachedUser = null;
  if (raw) {
    try {
      cachedUser = JSON.parse(raw) as UserPublic;
    } catch {
      cachedUser = null;
    }
  }
  return cachedUser;
}

const userListeners = new Set<() => void>();

function notifyUserChanged(): void {
  for (const listener of userListeners) listener();
}

/** Subscribes to session changes in this tab and via the storage event in others. */
export function subscribeToUser(listener: () => void): () => void {
  userListeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    userListeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

export function saveSession(user: UserPublic, tokens: TokenPair): void {
  setUser(user);
  setTokens(tokens);
}

export function clearSession(): void {
  clearTokens();
}

export function getUserRole(): Role | null {
  return getUser()?.role ?? null;
}

/** Clears local session and best-effort revokes the refresh token server-side. */
export async function logout(): Promise<void> {
  const refresh = getRefreshToken();
  clearSession();
  if (refresh) {
    // Bare fetch avoids an import cycle with lib/api.
    const base = (
      process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
    ).replace(/\/$/, "");
    try {
      await fetch(`${base}/api/v1/auth/logout`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refresh }),
      });
    } catch {
      /* best effort; the token still expires on its own */
    }
  }
  // Full document load clears in-memory data after sign-out.
  // eslint-disable-next-line @next/next/no-location-assign-relative-destination
  window.location.href = "/login";
}
