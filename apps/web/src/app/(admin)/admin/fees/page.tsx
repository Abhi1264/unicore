"use client";

import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
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

type FeeHead = {
  id: string;
  name: string;
  amount: number;
  due_date?: string | null;
  [key: string]: unknown;
};

export default function AdminFeesPage() {
  const [form, setForm] = useState({
    name: "",
    amount: "",
    due_date: "",
    late_fee_amount: "0",
  });
  const [formError, setFormError] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () => apiFetch<{ fee_heads: FeeHead[] }>("/api/v1/fees/heads"),
    [],
    "Failed to load fee heads.",
  );
  const heads = data?.fee_heads ?? [];
  const error = formError ?? loadError;

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    try {
      await apiFetch("/api/v1/fees/heads", {
        method: "POST",
        body: {
          name: form.name,
          amount: Number(form.amount),
          due_date: form.due_date || undefined,
          late_fee_amount: Number(form.late_fee_amount) || 0,
        },
      });
      setForm({ name: "", amount: "", due_date: "", late_fee_amount: "0" });
      reload();
    } catch (err) {
      setFormError(errorMessage(err, "Could not create the fee head."));
    }
  }

  return (
    <div>
      <PageHeader title="Fee heads" description="Define fee categories and amounts." />
      {error ? <ErrorBanner message={error} /> : null}
      <form onSubmit={create} className="mb-8 grid max-w-2xl gap-3 sm:grid-cols-2">
        <div className="space-y-2 sm:col-span-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            required
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="amount">Amount</Label>
          <Input
            id="amount"
            type="number"
            required
            value={form.amount}
            onChange={(e) => setForm({ ...form, amount: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="due_date">Due date</Label>
          <Input
            id="due_date"
            type="date"
            value={form.due_date}
            onChange={(e) => setForm({ ...form, due_date: e.target.value })}
          />
        </div>
        <Button type="submit" className="sm:col-span-2 sm:w-fit">
          Add fee head
        </Button>
      </form>
      {loading ? (
        <EmptyState message="Loading…" />
      ) : heads.length === 0 ? (
        <EmptyState message="No fee heads yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead className="text-right">Amount</TableHead>
                <TableHead>Due</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {heads.map((h) => (
                <TableRow key={h.id}>
                  <TableCell className="font-medium">{h.name}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {Number(h.amount).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {h.due_date ? String(h.due_date).slice(0, 10) : "—"}
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
