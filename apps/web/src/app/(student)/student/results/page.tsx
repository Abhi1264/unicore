"use client";

import { Suspense, use, useMemo, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { StudentResultsResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

type ResultsLoad =
  | { data: StudentResultsResponse; error: null }
  | { data: null; error: string };

function loadResults(): Promise<ResultsLoad> {
  return apiFetch<StudentResultsResponse>("/api/v1/results/me")
    .then((data) => ({ data, error: null }))
    .catch((err: unknown) => ({
      data: null,
      error:
        err instanceof ApiRequestError
          ? err.message
          : "Could not load results.",
    }));
}

function ResultsView({ resultsPromise }: { resultsPromise: Promise<ResultsLoad> }) {
  const result = use(resultsPromise);
  const [semester, setSemester] = useState("all");

  const semesterOptions = useMemo(() => {
    const set = new Set<string>();
    result.data?.rows.forEach((r) => {
      if (r.semester) set.add(r.semester);
    });
    return Array.from(set).sort((a, b) =>
      a.localeCompare(b, undefined, { numeric: true }),
    );
  }, [result.data]);

  if (result.error || !result.data) {
    return <ErrorBanner message={result.error ?? "Could not load results."} />;
  }

  const data = result.data;
  const filteredRows =
    semester === "all"
      ? data.rows
      : data.rows.filter((r) => r.semester === semester);

  return (
    <>
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <label
            htmlFor="semester"
            className="mb-1.5 block text-xs font-medium uppercase tracking-wide text-muted-foreground"
          >
            Semester
          </label>
          <select
            id="semester"
            value={semester}
            onChange={(e) => setSemester(e.target.value)}
            className="h-9 min-w-40 rounded-2xl border border-input bg-card px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
          >
            <option value="all">All semesters</option>
            {semesterOptions.map((s) => (
              <option key={s} value={s}>
                Semester {s}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-wrap gap-3">
          <div className="rounded-2xl border border-border bg-card px-5 py-3">
            <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              Cumulative
            </p>
            <p className="mt-0.5 text-2xl font-semibold tabular-nums tracking-tight text-teal">
              {data.cumulative_display || data.cumulative_value.toFixed(2)}
            </p>
            <p className="text-xs text-muted-foreground">
              {data.grading_system.replace(/_/g, " ")}
            </p>
          </div>
          <div className="rounded-2xl border border-border bg-card px-5 py-3">
            <p className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
              Courses
            </p>
            <p className="mt-0.5 text-2xl font-semibold tabular-nums tracking-tight text-ink">
              {filteredRows.length}
            </p>
            <p className="text-xs text-muted-foreground">
              {semester === "all" ? "all published" : `sem ${semester}`}
            </p>
          </div>
        </div>
      </div>

      {filteredRows.length === 0 ? (
        <EmptyState message="No published results for this filter yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Code</TableHead>
                <TableHead>Course</TableHead>
                <TableHead className="text-right">Credits</TableHead>
                <TableHead>Grade</TableHead>
                <TableHead className="text-right">Points</TableHead>
                <TableHead className="text-right">Marks</TableHead>
                <TableHead>Sem</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-mono text-xs">
                    {row.course_code}
                  </TableCell>
                  <TableCell className="max-w-56 truncate font-medium">
                    {row.course_name}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.credits}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={cn(
                        "font-semibold",
                        row.grade === "F" || row.grade === "Fail"
                          ? "bg-destructive/15 text-destructive"
                          : "bg-accent text-accent-foreground",
                      )}
                    >
                      {row.grade}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.grade_points != null ? row.grade_points.toFixed(1) : "—"}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.marks != null ? row.marks : "—"}
                  </TableCell>
                  <TableCell className="tabular-nums">{row.semester}</TableCell>
                  <TableCell className="text-xs capitalize text-muted-foreground">
                    {row.submission_status}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  );
}

export default function StudentResultsPage() {
  const [resultsPromise] = useState(loadResults);

  return (
    <div>
      <PageHeader
        title="Results"
        description="Published grades and cumulative standing for your program."
      />
      <Suspense fallback={<EmptyState message="Loading results…" />}>
        <ResultsView resultsPromise={resultsPromise} />
      </Suspense>
    </div>
  );
}
