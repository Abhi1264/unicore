"use client";

import { useState } from "react";
import { apiFetch, downloadFile } from "@/lib/api";
import { errorMessage, useAsyncData } from "@/lib/use-async-data";
import {
  DOCUMENT_TYPE_LABELS,
  type DocumentRequest,
  type DocumentType,
  type DocumentsResponse,
} from "@/lib/types";
import { EmptyState, ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const DOCUMENT_TYPES = Object.keys(DOCUMENT_TYPE_LABELS) as DocumentType[];

function statusVariant(status: DocumentRequest["status"]) {
  if (status === "ready") return "default" as const;
  if (status === "failed") return "destructive" as const;
  return "secondary" as const;
}

export default function StudentDocumentsPage() {
  const [docType, setDocType] = useState<DocumentType>("bonafide");
  const [actionError, setActionError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [downloading, setDownloading] = useState<string | null>(null);

  const {
    data,
    error: loadError,
    loading,
    reload,
  } = useAsyncData(
    () => apiFetch<DocumentsResponse>("/api/v1/documents"),
    [],
    "Failed to load documents.",
  );
  const docs = data?.documents ?? [];
  const error = actionError ?? loadError;

  async function requestDoc(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setActionError(null);
    try {
      // The API reads this field as `type`; it validates against a closed set
      // and rejects anything else.
      await apiFetch("/api/v1/documents", {
        method: "POST",
        body: { type: docType },
      });
      reload();
    } catch (err) {
      setActionError(errorMessage(err, "Could not submit the request."));
    } finally {
      setSubmitting(false);
    }
  }

  async function download(doc: DocumentRequest) {
    setDownloading(doc.id);
    setActionError(null);
    try {
      await downloadFile(
        `/api/v1/documents/${doc.id}/download`,
        `${doc.type ?? "document"}.pdf`,
      );
    } catch (err) {
      setActionError(errorMessage(err, "Could not download the document."));
    } finally {
      setDownloading(null);
    }
  }

  return (
    <div>
      <PageHeader
        title="Documents"
        description="Request certificates and download them once they are ready."
      />
      {error ? <ErrorBanner message={error} /> : null}

      <form
        onSubmit={requestDoc}
        className="mb-8 flex flex-wrap items-end gap-3"
      >
        <div className="space-y-2">
          <Label htmlFor="docType">Document type</Label>
          <Select
            value={docType}
            onValueChange={(value) => setDocType(value as DocumentType)}
          >
            <SelectTrigger id="docType" className="w-64">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {DOCUMENT_TYPES.map((t) => (
                <SelectItem key={t} value={t}>
                  {DOCUMENT_TYPE_LABELS[t]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Requesting…" : "Request"}
        </Button>
      </form>

      {loading ? (
        <EmptyState message="Loading documents…" />
      ) : docs.length === 0 ? (
        <EmptyState message="No document requests yet." />
      ) : (
        <div className="overflow-hidden rounded-2xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Type</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Requested</TableHead>
                <TableHead>
                  <span className="sr-only">Actions</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {docs.map((d) => (
                <TableRow key={d.id}>
                  <TableCell className="font-medium">
                    {d.type ? DOCUMENT_TYPE_LABELS[d.type] : "—"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(d.status)}>
                      {d.status ?? "requested"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {d.created_at
                      ? new Date(d.created_at).toLocaleDateString()
                      : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    {d.status === "ready" ? (
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={downloading === d.id}
                        onClick={() => void download(d)}
                      >
                        {downloading === d.id ? "Downloading…" : "Download"}
                      </Button>
                    ) : null}
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
