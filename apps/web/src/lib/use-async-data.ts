"use client";

import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { ApiRequestError } from "./api";

/**
 * Loads data for a client page; loading is derived by comparing settled key vs current key
 * (avoids synchronous setState-in-effect).
 */

export type AsyncState<T> = {
  data: T | null;
  error: string | null;
  loading: boolean;
};

export type AsyncResult<T> = AsyncState<T> & {
  /** Re-runs the loader, e.g. after a mutation. */
  reload: () => void;
  /** Replaces the current data without a round trip. */
  setData: (next: T) => void;
};

type Settled<T> =
  | { key: string; data: T; error: null }
  | { key: string; data: null; error: string };

export function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiRequestError) return err.message;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
}

export function useAsyncData<T>(
  load: () => Promise<T>,
  deps: readonly unknown[] = [],
  fallbackMessage = "Something went wrong. Please try again.",
): AsyncResult<T> {
  const [attempt, setAttempt] = useState(0);
  const [settled, setSettled] = useState<Settled<T> | null>(null);

  // Keep latest loader in a ref so the effect keys on deps, not a new function each render.
  const loadRef = useRef(load);
  useLayoutEffect(() => {
    loadRef.current = load;
  });

  const key = `${attempt}:${JSON.stringify(deps)}`;

  useEffect(() => {
    let cancelled = false;
    loadRef
      .current()
      .then((data) => {
        if (!cancelled) setSettled({ key, data, error: null });
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setSettled({ key, data: null, error: errorMessage(err, fallbackMessage) });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [key, fallbackMessage]);

  const reload = useCallback(() => setAttempt((n) => n + 1), []);
  const setData = useCallback(
    (next: T) => setSettled((prev) => ({ key: prev?.key ?? key, data: next, error: null })),
    [key],
  );

  const current = settled?.key === key ? settled : null;
  return {
    data: current?.data ?? null,
    error: current?.error ?? null,
    loading: current === null,
    reload,
    setData,
  };
}
