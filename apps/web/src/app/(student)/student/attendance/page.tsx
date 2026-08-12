"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { AttendanceSummary } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function StudentAttendancePage() {
  const [data, setData] = useState<AttendanceSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<AttendanceSummary>("/api/v1/attendance/summary");
        setData(res);
      } catch (err) {
        setError(
          err instanceof ApiRequestError ? err.message : "Failed to load attendance.",
        );
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const courses = data?.courses ?? [];

  return (
    <div>
      <PageHeader
        title="Attendance"
        description="Session-level summary across your enrolled courses."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {data?.overall_percentage != null ? (
        <p className="mb-6 text-sm text-muted-foreground">
          Overall:{" "}
          <span className="text-lg font-semibold tabular-nums text-teal">
            {Number(data.overall_percentage).toFixed(1)}%
          </span>
        </p>
      ) : null}
      {loading ? (
        <EmptyState message="Loading attendance…" />
      ) : courses.length === 0 ? (
        <EmptyState message="No attendance records yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Course</TableHead>
                <TableHead className="text-right">Present</TableHead>
                <TableHead className="text-right">Absent</TableHead>
                <TableHead className="text-right">%</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {courses.map((c) => (
                <TableRow key={c.course_id}>
                  <TableCell>
                    <span className="font-mono text-xs text-muted-foreground">
                      {c.course_code}
                    </span>
                    <span className="ml-2 font-medium">{c.course_name ?? c.course_id}</span>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{c.present}</TableCell>
                  <TableCell className="text-right tabular-nums">{c.absent}</TableCell>
                  <TableCell className="text-right font-medium tabular-nums text-teal">
                    {Number(c.percentage).toFixed(1)}%
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
