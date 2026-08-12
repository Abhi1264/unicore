"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { AuditLog, AuditLogsResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function AdminAuditPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<AuditLogsResponse>("/api/v1/admin/audit-logs");
        setLogs(res.logs ?? []);
      } catch (err) {
        setError(err instanceof ApiRequestError ? err.message : "Load failed.");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div>
      <PageHeader title="Audit log" description="Recent administrative actions." />
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <EmptyState message="Loading…" />
      ) : logs.length === 0 ? (
        <EmptyState message="No audit entries." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>When</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Resource</TableHead>
                <TableHead>Actor</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((l) => (
                <TableRow key={l.id}>
                  <TableCell className="text-sm text-muted-foreground">
                    {l.created_at
                      ? new Date(l.created_at).toLocaleString()
                      : "—"}
                  </TableCell>
                  <TableCell className="font-medium">{l.action ?? "—"}</TableCell>
                  <TableCell className="text-sm">{l.resource ?? "—"}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {l.actor_id ?? "—"}
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
