"use client";

import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type {
  AttendanceMark,
  AttendanceSessionResponse,
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
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const STATUSES = ["present", "absent", "late", "excused"] as const;
type Status = (typeof STATUSES)[number];

function studentKey(s: RosterStudent): string {
  return s.student_id || s.id || "";
}

export default function FacultyAttendancePage() {
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState("1");
  const [sessionDate, setSessionDate] = useState(
    () => new Date().toISOString().slice(0, 10),
  );
  const [marks, setMarks] = useState<Record<string, Status>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

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
      if (!courseId) return { roster: [] as RosterStudent[], existing: [] as AttendanceMark[] };
      const [rosterRes, sessionRes] = await Promise.all([
        apiFetch<RosterResponse>(
          `/api/v1/courses/${encodeURIComponent(courseId)}/roster?semester=${encodeURIComponent(semester)}`,
        ),
        apiFetch<AttendanceSessionResponse>(
          `/api/v1/attendance/session?course_id=${encodeURIComponent(courseId)}&session_date=${encodeURIComponent(sessionDate)}`,
        ).catch(() => ({ marks: [] })),
      ]);
      return {
        roster: rosterRes.roster ?? rosterRes.students ?? [],
        existing: sessionRes.marks ?? [],
      };
    },
    [courseId, semester, sessionDate],
    "Failed to load the roster.",
  );

  const roster = useMemo(() => sheet?.roster ?? [], [sheet]);
  const merged = useMemo(() => {
    const next: Record<string, Status> = {};
    for (const s of roster) {
      const id = studentKey(s);
      next[id] = "present";
    }
    for (const m of sheet?.existing ?? []) {
      next[m.student_id] = m.status;
    }
    return { ...next, ...marks };
  }, [roster, sheet?.existing, marks]);

  const error = formError ?? catalogError ?? sheetError;

  function setStatus(id: string, status: Status) {
    setMarks((prev) => ({ ...prev, [id]: status }));
    setOk(null);
  }

  function markAll(status: Status) {
    const next: Record<string, Status> = {};
    for (const s of roster) next[studentKey(s)] = status;
    setMarks(next);
    setOk(null);
  }

  async function save() {
    setFormError(null);
    setOk(null);
    setSaving(true);
    try {
      await apiFetch("/api/v1/attendance/session", {
        method: "POST",
        body: {
          course_id: courseId,
          session_date: sessionDate,
          marks: roster.map((s) => ({
            student_id: studentKey(s),
            status: merged[studentKey(s)] ?? "present",
          })),
        },
      });
      setOk(`Saved attendance for ${roster.length} students.`);
      setMarks({});
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not save attendance."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Mark attendance"
        description="Take the register for a class session. Everyone starts present — change only the exceptions."
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
              setMarks({});
              setOk(null);
            }}
            disabled={catalogLoading}
          />
        </div>
        <SemesterField value={semester} onChange={setSemester} />
        <div className="space-y-2">
          <Label htmlFor="session_date">Session date</Label>
          <Input
            id="session_date"
            type="date"
            required
            value={sessionDate}
            onChange={(e) => {
              setSessionDate(e.target.value);
              setMarks({});
              setOk(null);
            }}
          />
        </div>
      </div>

      {!courseId ? (
        <EmptyState message="Choose a course to load today’s register." />
      ) : sheetLoading ? (
        <EmptyState message="Loading roster…" />
      ) : roster.length === 0 ? (
        <EmptyState message="No students enrolled in this course for that semester." />
      ) : (
        <>
          <div className="mb-4 flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => markAll("present")}>
              All present
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => markAll("absent")}>
              All absent
            </Button>
          </div>
          <div className="overflow-hidden rounded-2xl border border-border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Roll</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {roster.map((s) => {
                  const id = studentKey(s);
                  return (
                    <TableRow key={id}>
                      <TableCell className="font-mono text-xs">
                        {s.roll_number ?? s.enrollment_number ?? "—"}
                      </TableCell>
                      <TableCell className="font-medium">{s.full_name ?? "—"}</TableCell>
                      <TableCell>
                        <select
                          aria-label={`Attendance for ${s.full_name ?? s.roll_number}`}
                          className="h-9 rounded-2xl border border-input bg-background px-3 text-sm"
                          value={merged[id] ?? "present"}
                          onChange={(e) => setStatus(id, e.target.value as Status)}
                        >
                          {STATUSES.map((st) => (
                            <option key={st} value={st}>
                              {st[0]!.toUpperCase() + st.slice(1)}
                            </option>
                          ))}
                        </select>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
          <Button className="mt-6" disabled={saving} onClick={() => void save()}>
            {saving ? "Saving…" : `Save session · ${roster.length}`}
          </Button>
        </>
      )}
    </div>
  );
}
