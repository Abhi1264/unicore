"use client";

import { useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { RosterResponse, RosterStudent } from "@/lib/types";
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

export default function FacultyRosterPage() {
  const [courseId, setCourseId] = useState("");
  const [students, setStudents] = useState<RosterStudent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function load(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<RosterResponse>(
        `/api/v1/courses/${encodeURIComponent(courseId)}/roster`,
      );
      setStudents(res.students ?? res.roster ?? []);
    } catch (err) {
      setStudents([]);
      setError(err instanceof ApiRequestError ? err.message : "Failed to load roster.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader title="Roster" description="Enrolled students for a course." />
      {error ? <ErrorBanner message={error} /> : null}
      <form onSubmit={load} className="mb-6 flex flex-wrap items-end gap-3">
        <div className="space-y-2">
          <Label htmlFor="courseId">Course ID</Label>
          <Input
            id="courseId"
            required
            value={courseId}
            onChange={(e) => setCourseId(e.target.value)}
            className="w-72"
          />
        </div>
        <Button type="submit" disabled={loading}>
          {loading ? "Loading…" : "Load roster"}
        </Button>
      </form>
      {students.length === 0 ? (
        <EmptyState message="Enter a course ID to view the roster." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Enrollment</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Email</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {students.map((s) => (
                <TableRow key={s.id}>
                  <TableCell className="font-mono text-xs">
                    {s.enrollment_number ?? s.id}
                  </TableCell>
                  <TableCell className="font-medium">{s.full_name ?? "—"}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {s.email ?? "—"}
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
