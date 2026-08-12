"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
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
  const [semester, setSemester] = useState("1");
  const [enrolling, setEnrolling] = useState<string | null>(null);
  const [enrollError, setEnrollError] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    async () => {
      // A closed registration window is an expected state rather than a
      // failure, so it must not take the course list down with it.
      const [courses, openWindow] = await Promise.all([
        apiFetch<CoursesResponse>("/api/v1/courses"),
        apiFetch<unknown>("/api/v1/registration-windows/open").catch(() => null),
      ]);
      return { courses: courses.courses ?? [], openWindow };
    },
    [],
    "Failed to load courses.",
  );
  const courses = data?.courses ?? [];
  const registrationOpen = Boolean(data?.openWindow);
  const error = enrollError ?? loadError;

  async function enroll(courseId: string) {
    setEnrolling(courseId);
    setEnrollError(null);
    try {
      await apiFetch("/api/v1/enrollments", {
        method: "POST",
        body: { course_id: courseId, semester },
        idempotencyKey: crypto.randomUUID(),
      });
      reload();
    } catch (err) {
      setEnrollError(errorMessage(err, "Enrollment could not be completed."));
    } finally {
      setEnrolling(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Course registration"
        description="Browse catalog courses and enroll while the window is open."
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
              {courses.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-mono text-xs">{c.code}</TableCell>
                  <TableCell className="font-medium">{c.name}</TableCell>
                  <TableCell className="text-right tabular-nums">{c.credits}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {c.seat_cap ?? "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      disabled={enrolling === c.id}
                      onClick={() => void enroll(c.id)}
                    >
                      {enrolling === c.id ? "Enrolling…" : "Enroll"}
                    </Button>
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
