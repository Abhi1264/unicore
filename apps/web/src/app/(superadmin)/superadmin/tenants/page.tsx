"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { TenantsQueueResponse } from "@/lib/types";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export default function SuperadminTenantsPage() {
  const [filter, setFilter] = useState<"pending" | "all">("pending");
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () =>
      apiFetch<TenantsQueueResponse>(
        `/api/v1/admin/tenants${filter === "pending" ? "?status=pending" : ""}`,
      ),
    [filter],
    "Failed to load tenants.",
  );
  const tenants = data?.tenants ?? [];
  const error = actionError ?? loadError;

  async function act(id: string, action: "approve" | "reject" | "suspend" | "reactivate") {
    setBusy(id);
    setActionError(null);
    try {
      await apiFetch(`/api/v1/admin/tenants/${id}/${action}`, { method: "POST" });
      reload();
    } catch (err) {
      setActionError(errorMessage(err, `Could not ${action} this tenant.`));
    } finally {
      setBusy(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Tenants queue"
        description="Approve or manage institute tenants on the platform."
      />
      {error ? <ErrorBanner message={error} /> : null}
      <div className="mb-6 flex gap-2">
        <Button
          size="sm"
          variant={filter === "pending" ? "default" : "outline"}
          onClick={() => setFilter("pending")}
        >
          Pending
        </Button>
        <Button
          size="sm"
          variant={filter === "all" ? "default" : "outline"}
          onClick={() => setFilter("all")}
        >
          All
        </Button>
      </div>
      {loading ? (
        <EmptyState message="Loading…" />
      ) : tenants.length === 0 ? (
        <EmptyState message="No tenants in this view." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Subdomain</TableHead>
                <TableHead>Status</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {tenants.map((t) => (
                <TableRow key={t.id}>
                  <TableCell className="font-medium">{t.name}</TableCell>
                  <TableCell className="font-mono text-xs">{t.subdomain}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{t.status}</Badge>
                  </TableCell>
                  <TableCell className="space-x-1 text-right">
                    {t.status === "pending" || t.status === "pending_approval" ? (
                      <>
                        <Button
                          size="xs"
                          disabled={busy === t.id}
                          onClick={() => act(t.id, "approve")}
                        >
                          Approve
                        </Button>
                        <Button
                          size="xs"
                          variant="outline"
                          disabled={busy === t.id}
                          onClick={() => act(t.id, "reject")}
                        >
                          Reject
                        </Button>
                      </>
                    ) : t.status === "active" ? (
                      <Button
                        size="xs"
                        variant="outline"
                        disabled={busy === t.id}
                        onClick={() => act(t.id, "suspend")}
                      >
                        Suspend
                      </Button>
                    ) : (
                      <Button
                        size="xs"
                        disabled={busy === t.id}
                        onClick={() => act(t.id, "reactivate")}
                      >
                        Reactivate
                      </Button>
                    )}
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
