"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse } from "@/lib/types";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { CourseSelect, DEFAULT_SEMESTER, SemesterField } from "@/components/course-select";
import { Button } from "@/components/ui/button";

export default function AdminPublishResultsPage() {
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState(DEFAULT_SEMESTER);
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const { data: catalog, loading: catalogLoading } = useAsyncData(
    () => apiFetch<CoursesResponse>("/api/v1/courses"),
    [],
    "Failed to load courses.",
  );
  const courses = catalog?.courses ?? [];

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setOk(null);
    try {
      await apiFetch("/api/v1/results/publish", {
        method: "POST",
        body: { course_id: courseId, semester },
      });
      setOk("Results published. Students can see them now.");
    } catch (err) {
      setError(errorMessage(err, "Publish failed."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Publish results"
        description="Release submitted grades for a course so students can view them."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="flex max-w-md flex-col gap-4">
        <CourseSelect
          courses={courses}
          value={courseId}
          onChange={setCourseId}
          disabled={catalogLoading}
        />
        <SemesterField value={semester} onChange={setSemester} />
        <Button type="submit" disabled={loading || !courseId}>
          {loading ? "Publishing…" : "Publish"}
        </Button>
      </form>
    </div>
  );
}
