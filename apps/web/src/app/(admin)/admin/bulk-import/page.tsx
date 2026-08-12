"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage } from "@/lib/use-async-data";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

export default function AdminBulkImportPage() {
  const [file, setFile] = useState<File | null>(null);
  const [jobType, setJobType] = useState("students");
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  // Remount file input after success (value cannot be cleared programmatically).
  const [inputGeneration, setInputGeneration] = useState(0);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      setError("Choose a CSV file.");
      return;
    }
    setLoading(true);
    setError(null);
    setOk(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("job_type", jobType);
      const data = await apiFetch<{ job?: { id?: string } }>(
        `/api/v1/admin/bulk-import?job_type=${encodeURIComponent(jobType)}`,
        { method: "POST", body: fd },
      );
      setOk(`Import queued${data.job?.id ? ` · job ${data.job.id}` : ""}.`);
      setFile(null);
      setInputGeneration((n) => n + 1);
    } catch (err) {
      setError(errorMessage(err, "Upload failed."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Bulk import"
        description="Upload CSV files for students or related entities."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="flex max-w-md flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="job_type">Job type</Label>
          <select
            id="job_type"
            className="h-9 w-full rounded-2xl border border-input bg-card px-3 text-sm"
            value={jobType}
            onChange={(e) => setJobType(e.target.value)}
          >
            {/* Keep in sync with API/worker job_type allow-list. */}
            <option value="students">Students</option>
            <option value="results">Results</option>
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="file">CSV file</Label>
          <input
            key={inputGeneration}
            id="file"
            type="file"
            accept=".csv,text/csv"
            className="block w-full text-sm"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </div>
        <Button type="submit" disabled={loading}>
          {loading ? "Uploading…" : "Upload"}
        </Button>
      </form>
    </div>
  );
}
