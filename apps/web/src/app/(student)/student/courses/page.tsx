"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse, Enrollment, EnrollmentsResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { DEFAULT_SEMESTER } from "@/components/course-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function StudentCoursesPage() {
  const [semester, setSemester] = useState(DEFAULT_SEMESTER);
  const [busy, setBusy] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    async () => {
      const [courses, mine, openWindow] = await Promise.all([
        apiFetch<CoursesResponse>("/api/v1/courses"),
        apiFetch<EnrollmentsResponse>("/api/v1/enrollments/me"),
        apiFetch<unknown>(
          `/api/v1/registration-windows/open?semester=${encodeURIComponent(semester)}`,
        ).catch(() => null),
      ]);
      return {
        courses: courses.courses ?? [],
        enrollments: mine.enrollments ?? [],
        openWindow,
      };
    },
    [semester],
    "Failed to load courses.",
  );
  const courses = data?.courses ?? [];
  const enrollments = data?.enrollments ?? [];
  const enrolledIds = new Set(
    enrollments.filter((e) => e.semester === semester).map((e) => e.course_id),
  );
  const registrationOpen = Boolean(data?.openWindow);
  const error = actionError ?? loadError;

  async function enroll(courseId: string) {
    setBusy(courseId);
    setActionError(null);
    try {
      await apiFetch("/api/v1/enrollments", {
        method: "POST",
        body: { course_id: courseId, semester },
        idempotencyKey: crypto.randomUUID(),
      });
      reload();
    } catch (err) {
      setActionError(errorMessage(err, "Enrollment could not be completed."));
    } finally {
      setBusy(null);
    }
  }

  async function drop(row: Enrollment) {
    setBusy(row.course_id);
    setActionError(null);
    try {
      await apiFetch("/api/v1/enrollments/drop", {
        method: "POST",
        body: { course_id: row.course_id, semester: row.semester },
      });
      reload();
    } catch (err) {
      setActionError(errorMessage(err, "Could not drop this course."));
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Course registration"
        description="Enroll while the window is open. Dropped seats return to the pool."
      />
      {error ? <ErrorBanner message={error} /> : null}

      <div className="mb-6 flex flex-wrap items-end gap-4">
        <div className="space-y-2">
          <Label htmlFor="semester">Semester</Label>
          <Input
            id="semester"
            value={semester}
            onChange={(e) => setSemester(e.target.value)}
            className="w-28"
          />
        </div>
        <p className="text-sm text-muted-foreground">
          {registrationOpen
            ? "Registration window is open."
            : "No open registration window detected."}
        </p>
      </div>

      <h2 className="mb-3 text-sm font-semibold tracking-tight text-foreground">
        Your courses
      </h2>
      {loading ? (
        <EmptyState message="Loading enrollments…" />
      ) : enrollments.length === 0 ? (
        <EmptyState message="You are not enrolled in any course yet." />
      ) : (
        <div className="mb-10 overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Code</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Semester</TableHead>
                <TableHead className="text-right">Credits</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {enrollments.map((e) => (
                <TableRow key={e.id}>
                  <TableCell className="font-mono text-xs">{e.course_code}</TableCell>
                  <TableCell className="font-medium">{e.course_name}</TableCell>
                  <TableCell>{e.semester}</TableCell>
                  <TableCell className="text-right tabular-nums">{e.credits}</TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={busy === e.course_id || !registrationOpen}
                      onClick={() => void drop(e)}
                    >
                      {busy === e.course_id ? "Dropping…" : "Drop"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <h2 className="mb-3 text-sm font-semibold tracking-tight text-foreground">
        Catalog
      </h2>
      {loading ? (
        <EmptyState message="Loading courses…" />
      ) : courses.length === 0 ? (
        <EmptyState message="No courses published yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Code</TableHead>
                <TableHead>Name</TableHead>
                <TableHead className="text-right">Credits</TableHead>
                <TableHead className="text-right">Seats</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {courses.map((c) => {
                const enrolled = enrolledIds.has(c.id);
                return (
                  <TableRow key={c.id}>
                    <TableCell className="font-mono text-xs">{c.code}</TableCell>
                    <TableCell className="font-medium">{c.name}</TableCell>
                    <TableCell className="text-right tabular-nums">{c.credits}</TableCell>
                    <TableCell className="text-right tabular-nums">
                      {c.seat_cap ?? "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      {enrolled ? (
                        <span className="text-xs text-muted-foreground">Enrolled</span>
                      ) : (
                        <Button
                          size="sm"
                          disabled={busy === c.id || !registrationOpen}
                          onClick={() => void enroll(c.id)}
                        >
                          {busy === c.id ? "Enrolling…" : "Enroll"}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
