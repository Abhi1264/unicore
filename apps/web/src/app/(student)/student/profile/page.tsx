"use client";

import type { UserPublic } from "@unicore/shared";
import { apiFetch } from "@/lib/api";
import { getUser, setUser } from "@/lib/auth";
import { useAsyncData } from "@/lib/use-async-data";
import {
  EmptyState,
  ErrorBanner,
  PageHeader,
} from "@/components/nav-shell";
import { Badge } from "@/components/ui/badge";

export default function StudentProfilePage() {
  const { data, error: loadError, loading } = useAsyncData(
    async () => {
      const me = await apiFetch<UserPublic>("/api/v1/auth/me");
      setUser(me);
      return me;
    },
    [],
    "Failed to load profile.",
  );

  // The cached copy from login lets the page render immediately and keeps it
  // usable offline, so a failed refresh is only surfaced when there is nothing
  // to fall back on.
  const cached = getUser();
  const user = data ?? cached;
  const error = cached ? null : loadError;

  if (loading && !user) return <EmptyState message="Loading profile…" />;

  return (
    <div>
      <PageHeader title="Profile" description="Your account on this institute." />
      {error ? <ErrorBanner message={error} /> : null}
      {!user ? (
        <EmptyState message="Not signed in." />
      ) : (
        <dl className="grid max-w-lg gap-4 text-sm">
          <div>
            <dt className="text-muted-foreground">Name</dt>
            <dd className="mt-0.5 text-base font-medium text-ink">{user.full_name}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Email</dt>
            <dd className="mt-0.5">{user.email}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Role</dt>
            <dd className="mt-1">
              <Badge variant="secondary">{user.role}</Badge>
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Status</dt>
            <dd className="mt-0.5">{user.is_active ? "Active" : "Inactive"}</dd>
          </div>
        </dl>
      )}
    </div>
  );
}
