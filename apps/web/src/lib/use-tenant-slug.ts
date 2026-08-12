"use client";

import { useSyncExternalStore } from "react";
import { tenantSlugFromLocation } from "./api";

// Slug is fixed for the document; nothing to subscribe to.
const subscribe = () => () => {};
const getSnapshot = () => tenantSlugFromLocation();
const getServerSnapshot = () => null;

/** Institute slug from the browser host (null during prerender). */
export function useTenantSlug(): string | null {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
