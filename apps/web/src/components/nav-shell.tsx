"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { Icon } from "@phosphor-icons/react";
import {
  CalendarBlank,
  ChartBar,
  ClipboardText,
  CurrencyInr,
  Exam,
  Files,
  GearSix,
  House,
  Megaphone,
  SignOut,
  Student,
  Table,
  UploadSimple,
  UsersThree,
  Buildings,
  Palette,
  Scroll,
} from "@phosphor-icons/react";
import type { Role } from "@unicore/shared";
import { cn } from "@/lib/utils";
import { logout } from "@/lib/auth";
import { useCurrentUser, useIsHydrated } from "@/lib/use-current-user";
import { homeForRole } from "@/lib/home";
import { Button } from "@/components/ui/button";

export type NavLink = { href: string; label: string; icon: Icon };

const ROLE_LINKS: Record<Role, NavLink[]> = {
  student: [
    { href: "/student", label: "Home", icon: House },
    { href: "/student/results", label: "Results", icon: Exam },
    { href: "/student/fees", label: "Fees", icon: CurrencyInr },
    { href: "/student/attendance", label: "Attendance", icon: ClipboardText },
    { href: "/student/courses", label: "Courses", icon: Student },
    { href: "/student/timetable", label: "Timetable", icon: Table },
    { href: "/student/announcements", label: "Announcements", icon: Megaphone },
    { href: "/student/documents", label: "Documents", icon: Files },
    { href: "/student/profile", label: "Profile", icon: GearSix },
  ],
  faculty: [
    { href: "/faculty", label: "Home", icon: House },
    { href: "/faculty/attendance", label: "Attendance", icon: ClipboardText },
    { href: "/faculty/results", label: "Results", icon: Exam },
    { href: "/faculty/roster", label: "Roster", icon: UsersThree },
    { href: "/faculty/announcements", label: "Announcements", icon: Megaphone },
  ],
  institute_admin: [
    { href: "/admin", label: "Home", icon: House },
    { href: "/admin/courses", label: "Courses", icon: Student },
    { href: "/admin/timetable", label: "Timetable", icon: Table },
    { href: "/admin/registration", label: "Registration", icon: CalendarBlank },
    { href: "/admin/fees", label: "Fees", icon: CurrencyInr },
    { href: "/admin/announcements", label: "Announcements", icon: Megaphone },
    { href: "/admin/publish-results", label: "Publish", icon: Exam },
    { href: "/admin/bulk-import", label: "Import", icon: UploadSimple },
    { href: "/admin/branding", label: "Branding", icon: Palette },
    { href: "/admin/audit", label: "Audit", icon: Scroll },
  ],
  superadmin: [
    { href: "/superadmin", label: "Home", icon: House },
    { href: "/superadmin/tenants", label: "Tenants", icon: Buildings },
    { href: "/superadmin/usage", label: "Usage", icon: ChartBar },
  ],
};

const ROLE_LABEL: Record<Role, string> = {
  student: "Student",
  faculty: "Faculty",
  institute_admin: "Admin",
  superadmin: "Platform",
};

function isActive(pathname: string, href: string) {
  if (href === "/student" || href === "/faculty" || href === "/admin" || href === "/superadmin") {
    return pathname === href;
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function NavShell({
  role,
  children,
}: {
  role: Role;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const links = ROLE_LINKS[role];
  const name = useCurrentUser()?.full_name ?? "";

  return (
    <div className="flex min-h-full flex-1 flex-col bg-background md:flex-row">
      <aside className="sticky top-0 z-20 flex w-full flex-col border-b border-sidebar-border bg-sidebar text-sidebar-foreground md:h-svh md:w-60 md:shrink-0 md:border-b-0 md:border-r">
        <div className="flex items-center justify-between gap-3 px-4 py-4 md:block md:px-5 md:py-6">
          <Link
            href={homeForRole(role)}
            className="block text-lg font-semibold tracking-tight text-sidebar-foreground transition-opacity hover:opacity-90"
          >
            Unicore
          </Link>
          <p className="hidden text-xs text-sidebar-foreground/65 md:mt-1 md:block">
            {ROLE_LABEL[role]} portal
          </p>
          <span className="rounded-md bg-sidebar-accent px-2 py-1 text-[11px] font-medium text-sidebar-accent-foreground md:hidden">
            {ROLE_LABEL[role]}
          </span>
        </div>

        <nav
          aria-label="Primary"
          className="flex flex-1 gap-1 overflow-x-auto px-2 pb-3 md:flex-col md:overflow-x-visible md:px-3 md:pb-4"
        >
          {links.map((link) => {
            const active = isActive(pathname, link.href);
            const Icon = link.icon;
            return (
              <Link
                key={link.href}
                href={link.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "inline-flex min-h-11 min-w-11 cursor-pointer items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors duration-200",
                  active
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-sidebar-foreground/80 hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground",
                )}
              >
                <Icon weight={active ? "fill" : "regular"} className="size-5 shrink-0" aria-hidden />
                <span className="whitespace-nowrap">{link.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="mt-auto border-t border-sidebar-border px-3 py-4">
          {name ? (
            <p className="mb-2 truncate px-2 text-xs text-sidebar-foreground/65" title={name}>
              {name}
            </p>
          ) : null}
          <Button
            variant="ghost"
            size="sm"
            className="h-11 w-full cursor-pointer justify-start gap-2 text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            onClick={() => void logout()}
          >
            <SignOut className="size-5" aria-hidden />
            Sign out
          </Button>
        </div>
      </aside>

      <main className="flex min-w-0 flex-1 flex-col">
        <div className="unicore-surface mx-auto w-full max-w-6xl flex-1 px-4 py-6 md:px-8 md:py-8">
          {children}
        </div>
      </main>
    </div>
  );
}

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description?: string;
  actions?: React.ReactNode;
}) {
  return (
    <header className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-foreground md:text-3xl">
          {title}
        </h1>
        {description ? (
          <p className="mt-1.5 max-w-2xl text-sm leading-relaxed text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      {actions ? <div className="flex flex-wrap gap-2">{actions}</div> : null}
    </header>
  );
}

export function EmptyState({
  message,
  action,
}: {
  message: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="rounded-2xl border border-dashed border-border bg-muted/40 px-4 py-12 text-center">
      <p className="text-sm text-muted-foreground">{message}</p>
      {action ? <div className="mt-4 flex justify-center">{action}</div> : null}
    </div>
  );
}

export function ErrorBanner({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="mb-4 rounded-2xl border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      {message}
    </div>
  );
}

export type QuickLink = {
  href: string;
  title: string;
  description: string;
  icon: Icon;
};

export function QuickLinkGrid({ links }: { links: QuickLink[] }) {
  return (
    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {links.map((link, i) => {
        const Icon = link.icon;
        return (
          <li
            key={link.href}
            className="animate-in fade-in slide-in-from-bottom-2 fill-mode-both"
            style={{ animationDelay: `${i * 40}ms`, animationDuration: "280ms" }}
          >
            <Link
              href={link.href}
              className="group flex h-full min-h-28 cursor-pointer flex-col rounded-2xl border border-border bg-card p-4 transition-colors duration-200 hover:border-primary/35 hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/35"
            >
              <Icon
                className="size-6 text-primary transition-transform duration-200 group-hover:scale-105"
                weight="duotone"
                aria-hidden
              />
              <span className="mt-3 text-sm font-semibold text-foreground">{link.title}</span>
              <span className="mt-1 text-xs leading-relaxed text-muted-foreground">
                {link.description}
              </span>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

export function DashboardGreeting({
  fallback = "there",
  subtitle,
}: {
  fallback?: string;
  subtitle: string;
}) {
  const fullName = useCurrentUser()?.full_name?.trim();
  const name = fullName ? (fullName.split(" ")[0] ?? fullName) : fallback;

  // Clock-based greeting waits for hydration to avoid SSR mismatch.
  const hydrated = useIsHydrated();
  const hour = hydrated ? new Date().getHours() : -1;
  const hello =
    hour < 0
      ? "Welcome"
      : hour < 12
        ? "Good morning"
        : hour < 17
          ? "Good afternoon"
          : "Good evening";

  return (
    <header className="mb-8">
      <p className="text-xs font-medium uppercase tracking-[0.14em] text-muted-foreground">
        Overview
      </p>
      <h1 className="mt-2 text-3xl font-semibold tracking-tight text-foreground md:text-4xl">
        {hello}, {name}
      </h1>
      <p className="mt-2 max-w-xl text-sm leading-relaxed text-muted-foreground">{subtitle}</p>
    </header>
  );
}
