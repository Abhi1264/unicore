"use client";

import { useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function FacultyAttendancePage() {
  const [form, setForm] = useState({
    student_id: "",
    course_id: "",
    session_date: new Date().toISOString().slice(0, 10),
    status: "present",
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
      await apiFetch("/api/v1/attendance", {
        method: "POST",
        body: form,
      });
      setOk("Attendance marked.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Failed to mark.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Mark attendance"
        description="Record a session for a student in your course."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="flex max-w-md flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="student_id">Student ID</Label>
          <Input
            id="student_id"
            required
            value={form.student_id}
            onChange={(e) => setForm({ ...form, student_id: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="course_id">Course ID</Label>
          <Input
            id="course_id"
            required
            value={form.course_id}
            onChange={(e) => setForm({ ...form, course_id: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="session_date">Session date</Label>
          <Input
            id="session_date"
            type="date"
            required
            value={form.session_date}
            onChange={(e) => setForm({ ...form, session_date: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="status">Status</Label>
          <select
            id="status"
            className="h-9 w-full rounded-2xl border border-input bg-card px-3 text-sm"
            value={form.status}
            onChange={(e) => setForm({ ...form, status: e.target.value })}
          >
            <option value="present">Present</option>
            <option value="absent">Absent</option>
            <option value="late">Late</option>
            <option value="excused">Excused</option>
          </select>
        </div>
        <Button type="submit" disabled={loading}>
          {loading ? "Saving…" : "Mark attendance"}
        </Button>
      </form>
    </div>
  );
}
