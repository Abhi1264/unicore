"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse, TimetableEntry, TimetableResponse } from "@/lib/types";
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

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function slotsFrom(res: TimetableResponse | TimetableEntry[] | null | undefined): TimetableEntry[] {
  if (!res) return [];
  if (Array.isArray(res)) return res;
  return res.slots ?? res.entries ?? res.timetable ?? [];
}

export default function AdminTimetablePage() {
  const [semester, setSemester] = useState("1");
  const [form, setForm] = useState({
    course_id: "",
    day_of_week: "1",
    start: "09:00",
    end: "10:00",
    room: "",
  });
  const [formError, setFormError] = useState<string | null>(null);

  const { data: catalog } = useAsyncData(
    () => apiFetch<CoursesResponse>("/api/v1/courses"),
    [],
    "Failed to load courses.",
  );
  const courses = catalog?.courses ?? [];

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () =>
      apiFetch<TimetableResponse>(
        `/api/v1/timetable?semester=${encodeURIComponent(semester)}`,
      ),
    [semester],
    "Failed to load timetable.",
  );
  const slots = slotsFrom(data);
  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    const [startHour, startMin] = form.start.split(":").map(Number);
    const [endHour, endMin] = form.end.split(":").map(Number);
    try {
      await apiFetch("/api/v1/timetable", {
        method: "POST",
        body: {
          course_id: form.course_id,
          semester,
          day_of_week: Number(form.day_of_week),
          start_hour: startHour,
          start_min: startMin,
          end_hour: endHour,
          end_min: endMin,
          room: form.room,
        },
      });
      setForm({ ...form, room: "" });
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not add the slot."));
    }
  }

  return (
    <div>
      <PageHeader
        title="Timetable"
        description="Place courses on the weekly grid. Students see this on their portal."
      />
      {error ? <ErrorBanner message={error} /> : null}

      <form
        onSubmit={create}
        className="mb-8 grid max-w-3xl gap-3 sm:grid-cols-2"
      >
        <div className="sm:col-span-2">
          <SemesterField value={semester} onChange={setSemester} />
        </div>
        <div className="sm:col-span-2">
          <CourseSelect
            courses={courses}
            value={form.course_id}
            onChange={(course_id) => setForm({ ...form, course_id })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="day">Day</Label>
          <select
            id="day"
            className="h-11 w-full rounded-2xl border border-input bg-card px-3 text-sm"
            value={form.day_of_week}
            onChange={(e) => setForm({ ...form, day_of_week: e.target.value })}
          >
            {DAYS.map((d, i) => (
              <option key={d} value={String(i)}>
                {d}
              </option>
            ))}
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="room">Room</Label>
          <Input
            id="room"
            value={form.room}
            onChange={(e) => setForm({ ...form, room: e.target.value })}
            placeholder="LT-1"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="start">Starts</Label>
          <Input
            id="start"
            type="time"
            value={form.start}
            onChange={(e) => setForm({ ...form, start: e.target.value })}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="end">Ends</Label>
          <Input
            id="end"
            type="time"
            value={form.end}
            onChange={(e) => setForm({ ...form, end: e.target.value })}
            required
          />
        </div>
        <Button type="submit" className="sm:col-span-2 sm:w-fit">
          Add slot
        </Button>
      </form>

      {loading ? (
        <EmptyState message="Loading timetable…" />
      ) : slots.length === 0 ? (
        <EmptyState message="No slots for this semester yet." />
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
              {slots.map((e) => {
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
                    <TableCell>{e.room || "—"}</TableCell>
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
