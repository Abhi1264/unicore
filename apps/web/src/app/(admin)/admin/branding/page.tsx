"use client";

import { useEffect, useState } from "react";
import { apiFetch, ApiRequestError } from "@/lib/api";
import { ErrorBanner, PageHeader } from "@/components/nav-shell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function AdminBrandingPage() {
  const [form, setForm] = useState({
    primary_color: "#0d3d3a",
    secondary_color: "#1a5c57",
    institute_display_name: "",
    logo_url: "",
  });
  const [error, setError] = useState<string | null>(null);
  const [ok, setOk] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [prefillFailed, setPrefillFailed] = useState(false);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch<{
          branding?: Record<string, string>;
          tenant?: { name?: string };
        }>("/api/v1/tenants/current");
        setForm((f) => ({
          ...f,
          primary_color: res.branding?.primary_color ?? f.primary_color,
          secondary_color: res.branding?.secondary_color ?? f.secondary_color,
          institute_display_name:
            res.branding?.institute_display_name ??
            res.tenant?.name ??
            f.institute_display_name,
          logo_url: res.branding?.logo_url ?? "",
        }));
      } catch {
        // Warn before save: form shows defaults, not stored branding.
        setPrefillFailed(true);
      }
    })();
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setOk(null);
    try {
      await apiFetch("/api/v1/tenants/current/branding", {
        method: "PATCH",
        body: {
          primary_color: form.primary_color,
          secondary_color: form.secondary_color || undefined,
          institute_display_name: form.institute_display_name || undefined,
          logo_url: form.logo_url || null,
        },
      });
      setOk("Branding updated.");
    } catch (err) {
      setError(err instanceof ApiRequestError ? err.message : "Update failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Branding"
        description="Institute colors and display name for the portal."
      />
      {error ? <ErrorBanner message={error} /> : null}
      {prefillFailed ? (
        <ErrorBanner message="Could not load your current branding, so these are default values. Saving will overwrite what is stored." />
      ) : null}
      {ok ? (
        <p className="mb-4 text-sm text-teal-soft" role="status">
          {ok}
        </p>
      ) : null}
      <form onSubmit={onSubmit} className="flex max-w-md flex-col gap-4">
        <div className="space-y-2">
          <Label htmlFor="institute_display_name">Display name</Label>
          <Input
            id="institute_display_name"
            value={form.institute_display_name}
            onChange={(e) =>
              setForm({ ...form, institute_display_name: e.target.value })
            }
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="primary_color">Primary color</Label>
          <Input
            id="primary_color"
            pattern="#[0-9A-Fa-f]{6}"
            value={form.primary_color}
            onChange={(e) => setForm({ ...form, primary_color: e.target.value })}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="secondary_color">Secondary color</Label>
          <Input
            id="secondary_color"
            pattern="#[0-9A-Fa-f]{6}"
            value={form.secondary_color}
            onChange={(e) =>
              setForm({ ...form, secondary_color: e.target.value })
            }
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="logo_url">Logo URL</Label>
          <Input
            id="logo_url"
            type="url"
            value={form.logo_url}
            onChange={(e) => setForm({ ...form, logo_url: e.target.value })}
          />
        </div>
        <Button type="submit" disabled={loading}>
          {loading ? "Saving…" : "Save branding"}
        </Button>
      </form>
    </div>
  );
}
