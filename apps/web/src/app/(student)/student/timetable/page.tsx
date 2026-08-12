"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { TimetableEntry, TimetableResponse } from "@/lib/types";
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

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export default function StudentTimetablePage() {
  const [entries, setEntries] = useState<TimetableEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<TimetableResponse | TimetableEntry[]>(
          "/api/v1/timetable",
        );
        if (Array.isArray(res)) setEntries(res);
        else setEntries(res.entries ?? res.timetable ?? []);
      } catch (err) {
        setError(
          err instanceof ApiRequestError ? err.message : "Failed to load timetable.",
        );
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div>
      <PageHeader title="Timetable" description="Weekly class schedule." />
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <EmptyState message="Loading timetable…" />
      ) : entries.length === 0 ? (
        <EmptyState message="No timetable entries." />
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
                      {e.start_time} – {e.end_time}
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
