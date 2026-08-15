"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { AnnouncementsResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

export function AnnouncementsPanel({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  const [headline, setHeadline] = useState("");
  const [body, setBody] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () => apiFetch<AnnouncementsResponse>("/api/v1/announcements"),
    [],
    "Failed to load announcements.",
  );
  const items = data?.announcements ?? [];
  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    try {
      await apiFetch("/api/v1/announcements", {
        method: "POST",
        body: { title: headline, body, audience_scope: "all" },
      });
      setHeadline("");
      setBody("");
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not publish the announcement."));
    }
  }

  return (
    <div>
      <PageHeader title={title} description={description} />
      {error ? <ErrorBanner message={error} /> : null}
      <form onSubmit={create} className="mb-8 flex max-w-lg flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="title">Title</Label>
          <Input
            id="title"
            required
            value={headline}
            onChange={(e) => setHeadline(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="body">Body</Label>
          <Textarea
            id="body"
            required
            rows={4}
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </div>
        <Button type="submit" className="w-fit">
          Publish
        </Button>
      </form>
      {loading ? (
        <EmptyState message="Loading…" />
      ) : items.length === 0 ? (
        <EmptyState message="No announcements yet." />
      ) : (
        <ul className="flex flex-col gap-4">
          {items.map((a) => (
            <li key={a.id} className="border-b border-border pb-4">
              <h2 className="font-medium">{a.title}</h2>
              <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">
                {a.body}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
