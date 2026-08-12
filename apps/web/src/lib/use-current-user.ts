"use client";

import { useSyncExternalStore } from "react";
import type { UserPublic } from "@unicore/shared";
import { getUser, subscribeToUser } from "./auth";

const getServerSnapshot = (): UserPublic | null => null;

/** Signed-in user from login cache (SSR-safe via useSyncExternalStore). */
export function useCurrentUser(): UserPublic | null {
  return useSyncExternalStore(subscribeToUser, getUser, getServerSnapshot);
}

const noopSubscribe = () => () => {};
const alwaysTrue = () => true;
const alwaysFalse = () => false;

/** False during prerender/first paint; true after hydration (clock/locale-safe). */
export function useIsHydrated(): boolean {
  return useSyncExternalStore(noopSubscribe, alwaysTrue, alwaysFalse);
}
