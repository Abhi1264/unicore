"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { Announcement, AnnouncementsResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";

export default function StudentAnnouncementsPage() {
  const [items, setItems] = useState<Announcement[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<AnnouncementsResponse>("/api/v1/announcements");
        setItems(res.announcements ?? []);
      } catch (err) {
        setError(
          err instanceof ApiRequestError
            ? err.message
            : "Failed to load announcements.",
        );
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div>
      <PageHeader
        title="Announcements"
        description="Campus notices for your audience."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <EmptyState message="Loading announcements…" />
      ) : items.length === 0 ? (
        <EmptyState message="No announcements yet." />
      ) : (
        <ul className="flex flex-col gap-4">
          {items.map((a) => (
            <li
              key={a.id}
              className="border-b border-border pb-4 last:border-0"
            >
              <h2 className="font-medium text-ink">{a.title}</h2>
              <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                {a.body}
              </p>
              {a.created_at ? (
                <p className="mt-2 text-xs text-muted-foreground">
                  {new Date(a.created_at).toLocaleString()}
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
