"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { RegistrationWindow } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { SemesterField } from "@/components/course-select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export default function AdminRegistrationPage() {
  const [semester, setSemester] = useState("1");
  const [form, setForm] = useState({
    name: "Course registration",
    opens_at: "",
    closes_at: "",
  });
  const [formError, setFormError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);

  const {
    data: openWindow,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () =>
      apiFetch<RegistrationWindow>(
        `/api/v1/registration-windows/open?semester=${encodeURIComponent(semester)}`,
      ).catch(() => null),
    [semester],
    "Failed to load the registration window.",
  );

  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    setOk(null);
    try {
      await apiFetch("/api/v1/registration-windows", {
        method: "POST",
        body: {
          name: form.name,
          semester,
          opens_at: new Date(form.opens_at).toISOString(),
          closes_at: new Date(form.closes_at).toISOString(),
        },
      });
      setOk("Registration window saved.");
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not open the window."));
    }
  }

  return (
    <div>
      <PageHeader
        title="Registration"
        description="Open a window so students can enroll. Outside these hours, enrollments are refused."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}

      <div className="mb-8 max-w-lg">
        <SemesterField value={semester} onChange={setSemester} />
      </div>

      {loading ? (
        <EmptyState message="Checking the current window…" />
      ) : openWindow?.id ? (
        <div className="mb-8 rounded-2xl border border-border bg-card px-4 py-4">
          <p className="text-sm font-medium text-foreground">{openWindow.name}</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Open {toLocalInput(openWindow.opens_at).replace("T", " ")} through{" "}
            {toLocalInput(openWindow.closes_at).replace("T", " ")}
          </p>
        </div>
      ) : (
        <EmptyState message="No registration window is open for this semester." />
      )}

      <form onSubmit={create} className="mt-8 flex max-w-lg flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">Window name</Label>
          <Input
            id="name"
            required
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="opens">Opens</Label>
          <Input
            id="opens"
            type="datetime-local"
            required
            value={form.opens_at}
            onChange={(e) => setForm({ ...form, opens_at: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="closes">Closes</Label>
          <Input
            id="closes"
            type="datetime-local"
            required
            value={form.closes_at}
            onChange={(e) => setForm({ ...form, closes_at: e.target.value })}
          />
        </div>
        <Button type="submit" className="w-fit">
          Open window
        </Button>
      </form>
    </div>
  );
}
