"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { RegisterTenantRequestSchema } from "@unicore/shared";
import { apiFetch, ApiRequestError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ErrorBanner } from "@/components/nav-shell";

export default function SignupPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    institute_name: "",
    subdomain: "",
    admin_email: "",
    admin_full_name: "",
    admin_password: "",
    timezone: "Asia/Kolkata",
  });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  function update<K extends keyof typeof form>(key: K, value: string) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const parsed = RegisterTenantRequestSchema.safeParse(form);
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Invalid form");
      return;
    }
    setLoading(true);
    try {
      await apiFetch("/api/v1/auth/register-tenant", {
        method: "POST",
        body: parsed.data,
        auth: false,
      });
      setDone(true);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(
          err.requestId
            ? `${err.message} (${err.code ?? "error"} · ${err.requestId})`
            : err.message,
        );
      } else {
        setError("Registration failed. Use app.localhost as the platform host.");
      }
    } finally {
      setLoading(false);
    }
  }

  if (done) {
    return (
      <div className="flex min-h-full flex-1 flex-col unicore-grid">
        <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center px-6 py-16">
          <p className="text-2xl font-semibold tracking-tight text-ink">Unicore</p>
          <h1 className="mt-8 text-xl font-medium text-ink">Request submitted</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Your institute <strong>{form.subdomain}</strong> is pending approval.
            After activation, sign in at{" "}
            <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
              {form.subdomain}.localhost
            </code>
            .
          </p>
          <Button className="mt-8" render={<Link href="/login" />}>
            Go to sign in
          </Button>
          <Button
            className="mt-2"
            variant="ghost"
            onClick={() => router.push("/")}
          >
            Back home
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-1 flex-col unicore-grid">
      <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center px-6 py-16">
        <Link href="/" className="text-2xl font-semibold tracking-tight text-ink">
          Unicore
        </Link>
        <h1 className="mt-8 text-xl font-medium text-ink">Register your institute</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Platform registration — call this from{" "}
          <code className="rounded bg-muted px-1 text-xs">app.localhost</code>.
        </p>

        {error ? (
          <div className="mt-6">
            <ErrorBanner message={error} />
          </div>
        ) : null}

        <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-4">
          <div className="space-y-2">
            <Label htmlFor="institute_name">Institute name</Label>
            <Input
              id="institute_name"
              required
              value={form.institute_name}
              onChange={(e) => update("institute_name", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="subdomain">Subdomain</Label>
            <Input
              id="subdomain"
              required
              pattern="[a-z0-9]([a-z0-9-]*[a-z0-9])?"
              placeholder="mit"
              value={form.subdomain}
              onChange={(e) =>
                update("subdomain", e.target.value.toLowerCase())
              }
            />
            <p className="text-xs text-muted-foreground">
              Students will use {form.subdomain || "slug"}.localhost
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin_full_name">Admin full name</Label>
            <Input
              id="admin_full_name"
              required
              value={form.admin_full_name}
              onChange={(e) => update("admin_full_name", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin_email">Admin email</Label>
            <Input
              id="admin_email"
              type="email"
              required
              value={form.admin_email}
              onChange={(e) => update("admin_email", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin_password">Admin password</Label>
            <Input
              id="admin_password"
              type="password"
              required
              minLength={8}
              value={form.admin_password}
              onChange={(e) => update("admin_password", e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="timezone">Timezone</Label>
            <Input
              id="timezone"
              value={form.timezone}
              onChange={(e) => update("timezone", e.target.value)}
            />
          </div>
          <Button type="submit" size="lg" disabled={loading} className="mt-2">
            {loading ? "Submitting…" : "Submit registration"}
          </Button>
        </form>

        <p className="mt-8 text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link
            href="/login"
            className="font-medium text-teal underline-offset-4 hover:underline"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
