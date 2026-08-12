"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import type { UsageResponse, UsageRow } from "@/lib/types";
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

export default function SuperadminUsagePage() {
  const [rows, setRows] = useState<UsageRow[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<UsageResponse>("/api/v1/admin/usage");
        setRows(res.usage ?? []);
      } catch (err) {
        setError(err instanceof ApiRequestError ? err.message : "Load failed.");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  return (
    <div>
      <PageHeader
        title="Usage"
        description="Platform usage metrics across tenants."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <EmptyState message="Loading…" />
      ) : rows.length === 0 ? (
        <EmptyState message="No usage data." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tenant</TableHead>
                <TableHead>Metric</TableHead>
                <TableHead className="text-right">Value</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r, i) => (
                <TableRow key={`${r.tenant_id ?? i}-${r.metric ?? i}`}>
                  <TableCell>
                    {r.name ?? r.subdomain ?? r.tenant_id ?? "—"}
                  </TableCell>
                  <TableCell>{r.metric ?? "—"}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {r.value ?? "—"}
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
