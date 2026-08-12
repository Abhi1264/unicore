"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import type { FeeDue, FeeDuesResponse } from "@/lib/types";
import { EmptyState, ErrorBanner, PageHeader } from "@/components/nav-shell";
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

const currency = new Intl.NumberFormat(undefined, {
  style: "currency",
  currency: "INR",
  maximumFractionDigits: 0,
});

export default function StudentFeesPage() {
  const [paying, setPaying] = useState<string | null>(null);
  const [payError, setPayError] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () => apiFetch<FeeDuesResponse>("/api/v1/fees/dues"),
    [],
    "Failed to load dues.",
  );
  const dues = data?.dues ?? [];
  const error = payError ?? loadError;

  async function pay(due: FeeDue) {
    const id = due.fee_head.id;
    setPaying(id);
    setPayError(null);
    try {
      // The key makes a retry or a double click resolve to the same payment
      // rather than charging twice.
      await apiFetch("/api/v1/fees/pay", {
        method: "POST",
        body: { fee_head_id: id },
        idempotencyKey: crypto.randomUUID(),
      });
      reload();
    } catch (err) {
      setPayError(errorMessage(err, "Payment could not be started."));
    } finally {
      setPaying(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Fees"
        description="Outstanding dues for the current term."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {loading ? (
        <EmptyState message="Loading dues…" />
      ) : dues.length === 0 ? (
        <EmptyState message="No outstanding dues." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Fee</TableHead>
                <TableHead className="text-right">Amount due</TableHead>
                <TableHead>Due date</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {dues.map((d) => {
                const id = d.fee_head.id;
                return (
                  <TableRow key={id}>
                    <TableCell className="font-medium">
                      {d.fee_head.name}
                      {d.late_fee_applied ? (
                        <span className="ml-2 text-xs text-muted-foreground">
                          includes late fee
                        </span>
                      ) : null}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {currency.format(d.amount_due)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {d.fee_head.due_date
                        ? String(d.fee_head.due_date).slice(0, 10)
                        : "—"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={d.is_overdue ? "destructive" : "outline"}>
                        {d.is_overdue ? "Overdue" : "Due"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="sm"
                        disabled={paying === id}
                        onClick={() => void pay(d)}
                      >
                        {paying === id ? "Paying…" : "Pay"}
                      </Button>
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
