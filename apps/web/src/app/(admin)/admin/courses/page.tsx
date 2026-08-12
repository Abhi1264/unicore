"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { CoursesResponse } from "@/lib/types";
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

export default function AdminCoursesPage() {
  const [form, setForm] = useState({
    code: "",
    name: "",
    credits: "3",
    seat_cap: "60",
  });
  const [formError, setFormError] = useState<string | null>(null);

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
  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    try {
      await apiFetch("/api/v1/courses", {
        method: "POST",
        body: {
          code: form.code,
          name: form.name,
          credits: Number(form.credits),
          seat_cap: Number(form.seat_cap),
        },
      });
      setForm({ code: "", name: "", credits: "3", seat_cap: "60" });
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not create the course."));
    }
  }

  return (
    <div>
      <PageHeader title="Courses" description="Manage the course catalog." />
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
                <TableRow key={c.id}>
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
    </div>
  );
}
