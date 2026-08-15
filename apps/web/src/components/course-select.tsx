"use client";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { Course } from "@/lib/types";

/** Matches the seeded academic term so demo data shows up without extra typing. */
export const DEFAULT_SEMESTER = "2026S1";

const selectClass =
  "h-11 w-full rounded-2xl border border-input bg-card px-3 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30";

export function CourseSelect({
  courses,
  value,
  onChange,
  id = "course",
  label = "Course",
  disabled,
}: {
  courses: Course[];
  value: string;
  onChange: (id: string) => void;
  id?: string;
  label?: string;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <select
        id={id}
        className={selectClass}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
      >
        <option value="">Select a course</option>
        {courses.map((c) => (
          <option key={c.id} value={c.id}>
            {c.code} · {c.name}
          </option>
        ))}
      </select>
    </div>
  );
}

export function SemesterField({
  value,
  onChange,
  id = "semester",
}: {
  value: string;
  onChange: (v: string) => void;
  id?: string;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>Semester</Label>
      <Input
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-28"
        required
      />
    </div>
  );
}
