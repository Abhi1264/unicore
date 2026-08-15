"use client";

import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { Announcement, AnnouncementsResponse, CoursesResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { CourseSelect } from "@/components/course-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

const SCOPES = [
  { value: "all", label: "Everyone" },
  { value: "program", label: "Program" },
  { value: "batch", label: "Batch year" },
  { value: "course", label: "Course" },
] as const;

type Scope = (typeof SCOPES)[number]["value"];

function audienceLabel(a: Announcement, courseName?: string): string {
  switch (a.audience_scope) {
    case "program":
      return a.audience_filter?.program
        ? `Program · ${a.audience_filter.program}`
        : "Program";
    case "batch":
      return a.audience_filter?.batch_year
        ? `Batch ${a.audience_filter.batch_year}`
        : "Batch";
    case "course":
      return courseName ? `Course · ${courseName}` : "Course";
    default:
      return "Everyone";
  }
}

export function AnnouncementsPanel({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  const [headline, setHeadline] = useState("");
  const [body, setBody] = useState("");
  const [scope, setScope] = useState<Scope>("all");
  const [program, setProgram] = useState("");
  const [batchYear, setBatchYear] = useState("");
  const [courseId, setCourseId] = useState("");
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

  const { data: catalog } = useAsyncData(
    () => apiFetch<CoursesResponse>("/api/v1/courses"),
    [],
    "Failed to load courses.",
  );
  const courses = catalog?.courses ?? [];
  const courseById = useMemo(() => {
    const list = catalog?.courses ?? [];
    return Object.fromEntries(list.map((c) => [c.id, `${c.code} · ${c.name}`]));
  }, [catalog?.courses]);
  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    const audience_filter =
      scope === "program"
        ? { program }
        : scope === "batch"
          ? { batch_year: Number(batchYear) }
          : scope === "course"
            ? { course_id: courseId }
            : {};
    if (scope === "course" && !courseId) {
      setFormError("Choose a course.");
      return;
    }
    try {
      await apiFetch("/api/v1/announcements", {
        method: "POST",
        body: { title: headline, body, audience_scope: scope, audience_filter },
      });
      setHeadline("");
      setBody("");
      setProgram("");
      setBatchYear("");
      setCourseId("");
      setScope("all");
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
        <div className="space-y-2">
          <Label htmlFor="audience">Audience</Label>
          <select
            id="audience"
            className="h-11 w-full rounded-2xl border border-input bg-card px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
            value={scope}
            onChange={(e) => setScope(e.target.value as Scope)}
          >
            {SCOPES.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        </div>
        {scope === "program" ? (
          <div className="space-y-2">
            <Label htmlFor="program">Program</Label>
            <Input
              id="program"
              required
              placeholder="B.Tech CSE"
              value={program}
              onChange={(e) => setProgram(e.target.value)}
            />
          </div>
        ) : null}
        {scope === "batch" ? (
          <div className="space-y-2">
            <Label htmlFor="batch_year">Batch year</Label>
            <Input
              id="batch_year"
              type="number"
              required
              min={1990}
              max={2100}
              placeholder="2024"
              value={batchYear}
              onChange={(e) => setBatchYear(e.target.value)}
            />
          </div>
        ) : null}
        {scope === "course" ? (
          <CourseSelect
            courses={courses}
            value={courseId}
            onChange={setCourseId}
            id="announce-course"
          />
        ) : null}
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
              <div className="flex flex-wrap items-baseline gap-2">
                <h2 className="font-medium">{a.title}</h2>
                <span className="text-xs text-muted-foreground">
                  {audienceLabel(
                    a,
                    a.audience_filter?.course_id
                      ? courseById[a.audience_filter.course_id]
                      : undefined,
                  )}
                </span>
              </div>
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
