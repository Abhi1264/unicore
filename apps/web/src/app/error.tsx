"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";

/** Catches render/data errors below the root layout. */
export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // digest is the only handle on the server-side stack.
    console.error("Unhandled application error", error.digest ?? "", error);
  }, [error]);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">
        Something went wrong
      </h1>
      <p className="max-w-prose text-sm text-muted-foreground">
        The page could not be displayed. You can try again, or head back to your
        dashboard.
      </p>
      {error.digest ? (
        <p className="font-mono text-xs text-muted-foreground">
          Reference: {error.digest}
        </p>
      ) : null}
      <div className="mt-2 flex gap-3">
        <Button onClick={reset}>Try again</Button>
        {/* Full reload escapes a broken client router tree. */}
        {/* eslint-disable-next-line @next/next/no-location-assign-relative-destination */}
        <Button variant="outline" onClick={() => (window.location.href = "/")}>
          Go home
        </Button>
      </div>
    </main>
  );
}
