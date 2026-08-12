"use client";

import { useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function AdminPublishResultsPage() {
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setOk(null);
    try {
      await apiFetch("/api/v1/results/publish", {
        method: "POST",
        body: {
          course_id: courseId,
          semester: semester || undefined,
        },
      });
      setOk("Results published for the selected course.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Publish failed.");
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
      <form onSubmit={onSubmit} className="flex max-w-sm flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="course_id">Course ID</Label>
          <Input
            id="course_id"
            required
            value={courseId}
            onChange={(e) => setCourseId(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="semester">Semester (optional)</Label>
          <Input
            id="semester"
            value={semester}
            onChange={(e) => setSemester(e.target.value)}
            placeholder="e.g. 3"
          />
        </div>
        <Button type="submit" disabled={loading}>
          {loading ? "Publishing…" : "Publish"}
        </Button>
      </form>
    </div>
  );
}
