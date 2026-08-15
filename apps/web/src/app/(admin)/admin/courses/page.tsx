"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type {
  Course,
  CoursesResponse,
  FacultyResponse,
  InstructorsResponse,
} from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import {
  CourseSelect,
  DEFAULT_SEMESTER,
  SemesterField,
} from "@/components/course-select";
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

export default function AdminCoursesPage() {
  const [form, setForm] = useState({
    code: "",
    name: "",
    credits: "3",
    seat_cap: "60",
  });
  const [formError, setFormError] = useState<string | null>(null);
  const [courseId, setCourseId] = useState("");
  const [semester, setSemester] = useState(DEFAULT_SEMESTER);
  const [facultyId, setFacultyId] = useState("");
  const [assigning, setAssigning] = useState(false);
  const [removing, setRemoving] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () => apiFetch<CoursesResponse>("/api/v1/courses"),
    [],
    "Failed to load courses.",
  );
  const courses = data?.courses ?? [];

  const { data: facultyRes, error: facultyError } = useAsyncData(
    () => apiFetch<FacultyResponse>("/api/v1/faculty"),
    [],
    "Failed to load faculty.",
  );
  const faculty = facultyRes?.faculty ?? [];

  const {
    data: instructorsRes,
    error: instructorsError,
    loading: instructorsLoading,
    reload: reloadInstructors,
  } = useAsyncData(
    async () => {
      if (!courseId) return { instructors: [] };
      return apiFetch<InstructorsResponse>(
        `/api/v1/courses/${encodeURIComponent(courseId)}/instructors?semester=${encodeURIComponent(semester)}`,
      );
    },
    [courseId, semester],
    "Failed to load instructors.",
  );
  const instructors = instructorsRes?.instructors ?? [];
  const error = formError ?? loadError ?? facultyError ?? instructorsError;
  const assignedIds = new Set(instructors.map((i) => i.faculty_id));
  const availableFaculty = faculty.filter((f) => !assignedIds.has(f.id));

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    try {
      const created = await apiFetch<Course>("/api/v1/courses", {
        method: "POST",
        body: {
          code: form.code,
          name: form.name,
          credits: Number(form.credits),
          seat_cap: Number(form.seat_cap),
        },
      });
      setForm({ code: "", name: "", credits: "3", seat_cap: "60" });
      if (created?.id) setCourseId(created.id);
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not create the course."));
    }
  }

  async function assign(e: React.FormEvent) {
    e.preventDefault();
    if (!courseId || !facultyId) return;
    setFormError(null);
    setAssigning(true);
    try {
      await apiFetch(`/api/v1/courses/${encodeURIComponent(courseId)}/instructors`, {
        method: "POST",
        body: { faculty_id: facultyId, semester },
      });
      setFacultyId("");
      reloadInstructors();
    } catch (err) {
      setFormError(errorMessage(err, "Could not assign the instructor."));
    } finally {
      setAssigning(false);
    }
  }

  async function remove(id: string) {
    setFormError(null);
    setRemoving(id);
    try {
      await apiFetch(`/api/v1/courses/${encodeURIComponent(courseId)}/instructors`, {
        method: "DELETE",
        body: { faculty_id: id, semester },
      });
      reloadInstructors();
    } catch (err) {
      setFormError(errorMessage(err, "Could not remove the instructor."));
    } finally {
      setRemoving(null);
    }
  }

  return (
    <div>
      <PageHeader title="Courses" description="Manage the course catalog and who teaches each offering." />
      {error ? <ErrorBanner message={error} /> : null}
      <form
        onSubmit={create}
        className="mb-8 grid max-w-2xl gap-3 sm:grid-cols-2"
      >
        <div className="space-y-2">
          <Label htmlFor="code">Code</Label>
          <Input
            id="code"
            required
            value={form.code}
            onChange={(e) => setForm({ ...form, code: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            required
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="credits">Credits</Label>
          <Input
            id="credits"
            type="number"
            step="0.5"
            value={form.credits}
            onChange={(e) => setForm({ ...form, credits: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="seat_cap">Seat cap</Label>
          <Input
            id="seat_cap"
            type="number"
            value={form.seat_cap}
            onChange={(e) => setForm({ ...form, seat_cap: e.target.value })}
          />
        </div>
        <Button type="submit" className="sm:col-span-2 sm:w-fit">
          Add course
        </Button>
      </form>
      {loading ? (
        <EmptyState message="Loading…" />
      ) : courses.length === 0 ? (
        <EmptyState message="No courses yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Code</TableHead>
                <TableHead>Name</TableHead>
                <TableHead className="text-right">Credits</TableHead>
                <TableHead className="text-right">Seats</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {courses.map((c) => (
                <TableRow
                  key={c.id}
                  className={c.id === courseId ? "bg-muted/60" : undefined}
                >
                  <TableCell className="font-mono text-xs">{c.code}</TableCell>
                  <TableCell>{c.name}</TableCell>
                  <TableCell className="text-right tabular-nums">{c.credits}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {c.seat_cap ?? "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <section className="mt-10 max-w-2xl">
        <h2 className="mb-1 text-lg font-medium">Instructors</h2>
        <p className="mb-4 text-sm text-muted-foreground">
          Faculty assigned here can mark attendance and enter grades for that course and semester.
        </p>
        <form onSubmit={assign} className="mb-6 grid gap-3 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <CourseSelect
              courses={courses}
              value={courseId}
              onChange={(id) => {
                setCourseId(id);
                setFacultyId("");
              }}
              disabled={loading}
            />
          </div>
          <SemesterField value={semester} onChange={setSemester} />
          <div className="space-y-2">
            <Label htmlFor="faculty">Faculty</Label>
            <select
              id="faculty"
              className="h-11 w-full rounded-2xl border border-input bg-card px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30"
              value={facultyId}
              onChange={(e) => setFacultyId(e.target.value)}
              disabled={!courseId}
            >
              <option value="">Select faculty</option>
              {availableFaculty.map((f) => (
                <option key={f.id} value={f.id}>
                  {f.full_name} · {f.email}
                </option>
              ))}
            </select>
          </div>
          <Button
            type="submit"
            className="sm:col-span-2 sm:w-fit"
            disabled={!courseId || !facultyId || assigning}
          >
            {assigning ? "Assigning…" : "Assign instructor"}
          </Button>
        </form>
        {!courseId ? (
          <EmptyState message="Choose a course to see who teaches it." />
        ) : instructorsLoading ? (
          <EmptyState message="Loading instructors…" />
        ) : instructors.length === 0 ? (
          <EmptyState message="No instructors assigned for this semester. Any faculty can mark until someone is assigned." />
        ) : (
          <ul className="flex flex-col gap-2">
            {instructors.map((i) => (
              <li
                key={i.id}
                className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-card px-4 py-3"
              >
                <div>
                  <p className="font-medium">{i.full_name}</p>
                  <p className="text-sm text-muted-foreground">{i.email}</p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  disabled={removing === i.faculty_id}
                  onClick={() => void remove(i.faculty_id)}
                >
                  {removing === i.faculty_id ? "Removing…" : "Remove"}
                </Button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
