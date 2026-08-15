"use client";

import { useEffect, useSyncExternalStore } from "react";
import type { UserPublic } from "@unicore/shared";
import { getUser, setUser, subscribeToUser } from "./auth";
import { apiFetch } from "./api";

const getServerSnapshot = (): UserPublic | null => null;

/** Signed-in user from login cache, hydrated from /auth/me when cookies exist. */
export function useCurrentUser(): UserPublic | null {
  const user = useSyncExternalStore(subscribeToUser, getUser, getServerSnapshot);

  useEffect(() => {
    if (user) return;
    let cancelled = false;
    void apiFetch<UserPublic>("/api/v1/auth/me")
      .then((me) => {
        if (!cancelled) setUser(me);
      })
      .catch(() => {
        /* no session cookie */
      });
    return () => {
      cancelled = true;
    };
  }, [user]);

  return user;
}

const noopSubscribe = () => () => {};
const alwaysTrue = () => true;
const alwaysFalse = () => false;

/** False during prerender/first paint; true after hydration (clock/locale-safe). */
export function useIsHydrated(): boolean {
  return useSyncExternalStore(noopSubscribe, alwaysTrue, alwaysFalse);
}
