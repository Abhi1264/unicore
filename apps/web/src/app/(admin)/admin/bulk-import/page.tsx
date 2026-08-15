"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { BulkJob, BulkJobsResponse } from "@/lib/types";
import { EmptyState, ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

function jobErrors(report: unknown): string[] {
  if (!report) return [];
  if (Array.isArray(report)) {
    return report.filter((x): x is string => typeof x === "string");
  }
  if (typeof report === "object" && report && "errors" in report) {
    const errs = (report as { errors: unknown }).errors;
    if (Array.isArray(errs)) {
      return errs.filter((x): x is string => typeof x === "string");
    }
  }
  return [];
}

function isTerminal(status: string): boolean {
  return status === "completed" || status === "failed";
}

export default function AdminBulkImportPage() {
  const [file, setFile] = useState<File | null>(null);
  const [jobType, setJobType] = useState("students");
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [activeJobId, setActiveJobId] = useState<string | null>(null);
  const [activeJob, setActiveJob] = useState<BulkJob | null>(null);
  // Remount file input after success (value cannot be cleared programmatically).
  const [inputGeneration, setInputGeneration] = useState(0);

  const {
    data,
    error: listError,
    loading: listLoading,
    reload,
  } = useAsyncData(
    () => apiFetch<BulkJobsResponse>("/api/v1/admin/bulk-jobs"),
    [],
    "Failed to load import jobs.",
  );
  const jobs = data?.jobs ?? [];
  const banner = error ?? listError;

  useEffect(() => {
    if (!activeJobId) return;
    const jobId = activeJobId;
    let cancelled = false;
    async function tick() {
      try {
        const job = await apiFetch<BulkJob>(
          `/api/v1/admin/bulk-jobs/${encodeURIComponent(jobId)}`,
        );
        if (cancelled) return;
        setActiveJob(job);
        if (isTerminal(job.status)) {
          setActiveJobId(null);
          reload();
          const errs = jobErrors(job.error_report);
          if (job.status === "failed") {
            setOk(null);
            setError(
              errs[0]
                ? `Import failed · ${errs[0]}`
                : "Import failed.",
            );
          } else {
            setError(null);
            setOk(
              `Import finished · ${job.success_rows}/${job.total_rows} rows succeeded.`,
            );
          }
        }
      } catch (err) {
        if (!cancelled) {
          setError(errorMessage(err, "Could not check import status."));
          setActiveJobId(null);
        }
      }
    }
    void tick();
    const id = window.setInterval(() => void tick(), 2000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [activeJobId, reload]);

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
      const data = await apiFetch<{ job?: BulkJob }>(
        `/api/v1/admin/bulk-import?job_type=${encodeURIComponent(jobType)}`,
        { method: "POST", body: fd },
      );
      const job = data.job;
      setFile(null);
      setInputGeneration((n) => n + 1);
      reload();
      if (job?.id) {
        setActiveJob(job);
        setActiveJobId(job.id);
        setOk(`Import queued · tracking job ${job.id.slice(0, 8)}…`);
      } else {
        setOk("Import queued.");
      }
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
        description="Upload CSV files for students or results, then watch the job finish."
      />
      {banner ? <ErrorBanner message={banner} /> : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      {activeJob && !isTerminal(activeJob.status) ? (
        <p className="mb-4 text-sm text-muted-foreground" role="status">
          Job {activeJob.id.slice(0, 8)} is {activeJob.status}
          {activeJob.total_rows
            ? ` · ${activeJob.success_rows}/${activeJob.total_rows} rows`
            : ""}.
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="mb-10 flex max-w-md flex-col gap-4">
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
        <Button type="submit" disabled={loading || Boolean(activeJobId)}>
          {loading ? "Uploading…" : activeJobId ? "Import running…" : "Upload"}
        </Button>
      </form>

      <h2 className="mb-3 text-lg font-medium">Recent jobs</h2>
      {listLoading && jobs.length === 0 ? (
        <EmptyState message="Loading jobs…" />
      ) : jobs.length === 0 ? (
        <EmptyState message="No import jobs yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Rows</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {jobs.map((j) => {
                const errs = jobErrors(j.error_report);
                return (
                  <TableRow key={j.id}>
                    <TableCell className="text-sm text-muted-foreground">
                      {j.created_at
                        ? new Date(j.created_at).toLocaleString()
                        : "—"}
                    </TableCell>
                    <TableCell className="capitalize">{j.job_type}</TableCell>
                    <TableCell>
                      <span className="capitalize">{j.status}</span>
                      {errs[0] ? (
                        <p className="mt-1 max-w-xs text-xs text-muted-foreground">
                          {errs[0]}
                          {errs.length > 1 ? ` (+${errs.length - 1} more)` : ""}
                        </p>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {j.total_rows
                        ? `${j.success_rows}/${j.total_rows}`
                        : "—"}
                    </TableCell>
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
