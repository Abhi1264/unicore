"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { useAsyncData } from "@/lib/use-async-data";
import type { TimetableEntry, TimetableResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { DEFAULT_SEMESTER, SemesterField } from "@/components/course-select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

function slotsFrom(res: TimetableResponse | TimetableEntry[] | null | undefined): TimetableEntry[] {
  if (!res) return [];
  if (Array.isArray(res)) return res;
  return res.slots ?? res.entries ?? res.timetable ?? [];
}

export default function StudentTimetablePage() {
  const [semester, setSemester] = useState(DEFAULT_SEMESTER);
  const { data, error, loading } = useAsyncData(
    () =>
      apiFetch<TimetableResponse>(
        `/api/v1/timetable?semester=${encodeURIComponent(semester)}`,
      ),
    [semester],
    "Failed to load timetable.",
  );
  const entries = slotsFrom(data);

  return (
    <div>
      <PageHeader title="Timetable" description="Your weekly class schedule." />
      {error ? <ErrorBanner message={error} /> : null}
      <div className="mb-6">
        <SemesterField value={semester} onChange={setSemester} />
      </div>
      {loading ? (
        <EmptyState message="Loading timetable…" />
      ) : entries.length === 0 ? (
        <EmptyState message="No timetable entries for this semester." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Day</TableHead>
                <TableHead>Time</TableHead>
                <TableHead>Course</TableHead>
                <TableHead>Room</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.map((e) => {
                const day =
                  typeof e.day_of_week === "number"
                    ? DAYS[e.day_of_week] ?? e.day_of_week
                    : e.day_of_week;
                return (
                  <TableRow key={e.id}>
                    <TableCell>{day}</TableCell>
                    <TableCell className="tabular-nums text-sm">
                      {String(e.start_time ?? "")} – {String(e.end_time ?? "")}
                    </TableCell>
                    <TableCell className="font-medium">
                      {e.course_code
                        ? `${e.course_code} · ${e.course_name ?? ""}`
                        : e.course_name ?? e.course_id}
                    </TableCell>
                    <TableCell>{e.room ?? "—"}</TableCell>
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
