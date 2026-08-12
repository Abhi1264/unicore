"use client";

import { useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function FacultyResultsPage() {
  const [form, setForm] = useState({
    student_id: "",
    course_id: "",
    semester: "1",
    grade: "",
    grade_points: "",
    marks: "",
    status: "submitted",
  });
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setOk(null);
    setLoading(true);
    try {
      await apiFetch("/api/v1/results", {
        method: "POST",
        body: {
          student_id: form.student_id,
          course_id: form.course_id,
          semester: form.semester,
          grade: form.grade,
          grade_points: form.grade_points
            ? Number(form.grade_points)
            : undefined,
          marks: form.marks ? Number(form.marks) : undefined,
          status: form.status,
        },
      });
      setOk("Result entered.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to enter.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Enter results"
        description="Submit grades for students in your courses."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="flex max-w-md flex-col gap-4">
        {(
          [
            ["student_id", "Student ID"],
            ["course_id", "Course ID"],
            ["semester", "Semester"],
            ["grade", "Grade"],
            ["grade_points", "Grade points"],
            ["marks", "Marks"],
          ] as const
        ).map(([key, label]) => (
          <div key={key} className="space-y-2">
            <Label htmlFor={key}>{label}</Label>
            <Input
              id={key}
              required={key === "student_id" || key === "course_id" || key === "grade"}
              value={form[key]}
              onChange={(e) => setForm({ ...form, [key]: e.target.value })}
            />
          </div>
        ))}
        <Button type="submit" disabled={loading}>
          {loading ? "Saving…" : "Submit result"}
        </Button>
      </form>
    </div>
  );
}
