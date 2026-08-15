"use client";

import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type {
  CourseResultRow,
  CourseResultsResponse,
  CoursesResponse,
  RosterResponse,
  RosterStudent,
} from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { CourseSelect, SemesterField } from "@/components/course-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type Draft = {
  grade: string;
  marks: string;
};

function studentKey(s: RosterStudent): string {
  return s.student_id || s.id || "";
}

export default function FacultyResultsPage() {
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState("1");
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [saving, setSaving] = useState<"draft" | "submitted" | null>(null);

  const { data: catalog, error: catalogError, loading: catalogLoading } =
    useAsyncData(
      () => apiFetch<CoursesResponse>("/api/v1/courses"),
      [],
      "Failed to load courses.",
    );
  const courses = catalog?.courses ?? [];

  const {
    data: sheet,
    error: sheetError,
    loading: sheetLoading,
    reload,
  } = useAsyncData(
    async () => {
      if (!courseId) {
        return { roster: [] as RosterStudent[], results: [] as CourseResultRow[] };
      }
      const [rosterRes, resultsRes] = await Promise.all([
        apiFetch<RosterResponse>(
          `/api/v1/courses/${encodeURIComponent(courseId)}/roster?semester=${encodeURIComponent(semester)}`,
        ),
        apiFetch<CourseResultsResponse>(
          `/api/v1/results/course?course_id=${encodeURIComponent(courseId)}&semester=${encodeURIComponent(semester)}`,
        ).catch(() => ({ results: [] })),
      ]);
      return {
        roster: rosterRes.roster ?? rosterRes.students ?? [],
        results: resultsRes.results ?? [],
      };
    },
    [courseId, semester],
    "Failed to load the grade sheet.",
  );

  const roster = useMemo(() => sheet?.roster ?? [], [sheet]);
  const merged = useMemo(() => {
    const byStudent = new Map((sheet?.results ?? []).map((r) => [r.student_id, r]));
    const next: Record<string, Draft> = {};
    for (const s of roster) {
      const id = studentKey(s);
      const existing = byStudent.get(id);
      next[id] = {
        grade: existing?.grade ?? "",
        marks:
          existing?.marks === null || existing?.marks === undefined
            ? ""
            : String(existing.marks),
      };
    }
    return { ...next, ...drafts };
  }, [roster, sheet?.results, drafts]);

  const error = formError ?? catalogError ?? sheetError;

  function patch(id: string, field: keyof Draft, value: string) {
    setDrafts((prev) => ({
      ...prev,
      [id]: { ...(merged[id] ?? { grade: "", marks: "" }), [field]: value },
    }));
    setOk(null);
  }

  async function save(status: "draft" | "submitted") {
    setFormError(null);
    setOk(null);
    const rows = roster
      .map((s) => {
        const id = studentKey(s);
        const d = merged[id];
        if (!d?.grade.trim()) return null;
        return {
          student_id: id,
          grade: d.grade.trim(),
          marks: d.marks ? Number(d.marks) : undefined,
        };
      })
      .filter((row): row is NonNullable<typeof row> => row !== null);

    if (rows.length === 0) {
      setFormError("Enter at least one grade before saving.");
      return;
    }
    setSaving(status);
    try {
      await apiFetch("/api/v1/results/batch", {
        method: "POST",
        body: { course_id: courseId, semester, status, rows },
      });
      setOk(
        status === "draft"
          ? `Saved ${rows.length} draft grades.`
          : `Submitted ${rows.length} grades for publish.`,
      );
      setDrafts({});
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not save grades."));
    } finally {
      setSaving(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Enter results"
        description="Grade the roster. Drafts stay internal until an admin publishes the course."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}

      <div className="mb-6 grid max-w-3xl gap-4 sm:grid-cols-3">
        <div className="sm:col-span-2">
          <CourseSelect
            courses={courses}
            value={courseId}
            onChange={(id) => {
              setCourseId(id);
              setDrafts({});
              setOk(null);
            }}
            disabled={catalogLoading}
          />
        </div>
        <SemesterField value={semester} onChange={setSemester} />
      </div>

      {!courseId ? (
        <EmptyState message="Choose a course to open the grade sheet." />
      ) : sheetLoading ? (
        <EmptyState message="Loading roster…" />
      ) : roster.length === 0 ? (
        <EmptyState message="No students enrolled in this course for that semester." />
      ) : (
        <>
          <div className="overflow-hidden rounded-2xl border border-border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Roll</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Grade</TableHead>
                  <TableHead>Marks</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {roster.map((s) => {
                  const id = studentKey(s);
                  const d = merged[id] ?? { grade: "", marks: "" };
                  return (
                    <TableRow key={id}>
                      <TableCell className="font-mono text-xs">
                        {s.roll_number ?? "—"}
                      </TableCell>
                      <TableCell className="font-medium">{s.full_name ?? "—"}</TableCell>
                      <TableCell>
                        <Input
                          aria-label={`Grade for ${s.full_name ?? s.roll_number}`}
                          value={d.grade}
                          onChange={(e) => patch(id, "grade", e.target.value)}
                          className="w-24"
                          placeholder="A"
                        />
                      </TableCell>
                      <TableCell>
                        <Input
                          aria-label={`Marks for ${s.full_name ?? s.roll_number}`}
                          type="number"
                          value={d.marks}
                          onChange={(e) => patch(id, "marks", e.target.value)}
                          className="w-24"
                        />
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <Button
              variant="outline"
              disabled={saving !== null}
              onClick={() => void save("draft")}
            >
              {saving === "draft" ? "Saving…" : "Save drafts"}
            </Button>
            <Button disabled={saving !== null} onClick={() => void save("submitted")}>
              {saving === "submitted" ? "Submitting…" : "Submit for publish"}
            </Button>
          </div>
        </>
      )}
    </div>
  );
}
