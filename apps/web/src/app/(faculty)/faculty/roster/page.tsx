"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse, RosterResponse, RosterStudent } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { CourseSelect, SemesterField } from "@/components/course-select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function FacultyRosterPage() {
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState("1");

  const { data: catalog, error: catalogError, loading: catalogLoading } =
    useAsyncData(
      () => apiFetch<CoursesResponse>("/api/v1/courses"),
      [],
      "Failed to load courses.",
    );
  const courses = catalog?.courses ?? [];

  const {
    data,
    error: rosterError,
    loading,
  } = useAsyncData(
    async () => {
      if (!courseId) return { students: [] as RosterStudent[] };
      const res = await apiFetch<RosterResponse>(
        `/api/v1/courses/${encodeURIComponent(courseId)}/roster?semester=${encodeURIComponent(semester)}`,
      );
      return { students: res.roster ?? res.students ?? [] };
    },
    [courseId, semester],
    "Failed to load roster.",
  );
  const students = data?.students ?? [];
  const error = catalogError ?? rosterError;

  return (
    <div>
      <PageHeader
        title="Roster"
        description="Who is enrolled, by course and semester."
      />
      {error ? <ErrorBanner message={error} /> : null}
      <div className="mb-6 grid max-w-3xl gap-4 sm:grid-cols-3">
        <div className="sm:col-span-2">
          <CourseSelect
            courses={courses}
            value={courseId}
            onChange={setCourseId}
            disabled={catalogLoading}
          />
        </div>
        <SemesterField value={semester} onChange={setSemester} />
      </div>
      {!courseId ? (
        <EmptyState message="Choose a course to view the roster." />
      ) : loading ? (
        <EmptyState message="Loading roster…" />
      ) : students.length === 0 ? (
        <EmptyState message="No students enrolled for that semester." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Roll</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Program</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {students.map((s) => (
                <TableRow key={s.student_id || s.id}>
                  <TableCell className="font-mono text-xs">
                    {s.roll_number ?? s.enrollment_number ?? "—"}
                  </TableCell>
                  <TableCell className="font-medium">{s.full_name ?? "—"}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {s.email ?? "—"}
                  </TableCell>
                  <TableCell className="text-sm">
                    {s.program ?? "—"}
                    {s.batch_year ? ` · ${s.batch_year}` : ""}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
