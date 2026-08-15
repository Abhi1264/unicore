import type { Role, UserPublic } from "@unicore/shared";

const USER_KEY = "unicore_user";

export function setUser(user: UserPublic): void {
  localStorage.setItem(USER_KEY, JSON.stringify(user));
  notifyUserChanged();
}

export function clearSession(): void {
  localStorage.removeItem(USER_KEY);
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

export function saveSession(user: UserPublic): void {
  setUser(user);
}

export function getUserRole(): Role | null {
  return getUser()?.role ?? null;
}

/** Clears local profile cache and revokes the httpOnly refresh cookie. */
export async function logout(): Promise<void> {
  clearSession();
  const base = (
    process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
  ).replace(/\/$/, "");
  try {
    await fetch(`${base}/api/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
    });
  } catch {
    /* best effort; the cookie still expires on its own */
  }
  // Full document load clears in-memory data after sign-out.
  // eslint-disable-next-line @next/next/no-location-assign-relative-destination
  window.location.href = "/login";
}
