"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { LoginRequestSchema, LoginResponseSchema } from "@unicore/shared";
import { apiFetch, ApiRequestError, platformOrigin } from "@/lib/api";
import { useTenantSlug } from "@/lib/use-tenant-slug";
import { saveSession } from "@/lib/auth";
import { homeForRole } from "@/lib/home";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ErrorBanner } from "@/components/nav-shell";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const slug = useTenantSlug();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const parsed = LoginRequestSchema.safeParse({ email, password });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Check your email and password");
      return;
    }
    if (!slug) {
      setError(
        "Open your institute's own address to sign in — accounts live on that subdomain, not here.",
      );
      return;
    }
    setLoading(true);
    try {
      const raw = await apiFetch<unknown>("/api/v1/auth/login", {
        method: "POST",
        body: parsed.data,
        auth: false,
      });
      const data = LoginResponseSchema.parse(raw);
      saveSession(data.user, data.tokens);
      router.push(homeForRole(data.user.role));
    } catch (err) {
      if (err instanceof ApiRequestError) {
        setError(
          err.code === "TENANT_REQUIRED"
            ? "Open your institute subdomain host, then try again."
            : err.message,
        );
      } else {
        setError("Unable to sign in. Try again in a moment.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-full flex-1 flex-col unicore-surface">
      <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center px-6 py-16">
        <Link
          href="/"
          className="text-2xl font-semibold tracking-tight text-foreground transition-opacity hover:opacity-80"
        >
          Unicore
        </Link>
        <h1 className="mt-10 text-2xl font-semibold tracking-tight text-foreground">
          Sign in
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
          {slug ? (
            <>
              Signing in to <span className="font-medium text-foreground">{slug}</span>
            </>
          ) : (
            "Open your institute's own address to sign in — accounts live on that subdomain."
          )}
        </p>

        {error ? (
          <div className="mt-6">
            <ErrorBanner message={error} />
          </div>
        ) : null}

        <form onSubmit={onSubmit} className="mt-8 flex flex-col gap-5">
          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="h-11"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Password</Label>
            <Input
              id="password"
              type="password"
              autoComplete="current-password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="h-11"
            />
          </div>
          <Button type="submit" size="lg" disabled={loading} className="mt-1 h-11 cursor-pointer">
            {loading ? "Signing in…" : "Sign in"}
          </Button>
        </form>

        <p className="mt-10 text-sm text-muted-foreground">
          New institute?{" "}
          <a
            href={`${platformOrigin()}/signup`}
            className="font-medium text-primary underline-offset-4 hover:underline"
          >
            Register on the platform
          </a>
        </p>
      </div>
    </div>
  );
}
