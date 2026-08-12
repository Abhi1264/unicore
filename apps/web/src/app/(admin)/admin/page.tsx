"use client";

import {
  Exam,
  Megaphone,
  Palette,
  Scroll,
  Student,
  UploadSimple,
  CurrencyInr,
} from "@phosphor-icons/react";
import {
  DashboardGreeting,
  QuickLinkGrid,
  type QuickLink,
} from "@/components/nav-shell";

const links: QuickLink[] = [
  {
    href: "/admin/courses",
    title: "Courses",
    description: "Departments, seat caps, and structure",
    icon: Student,
  },
  {
    href: "/admin/fees",
    title: "Fee heads",
    description: "Amounts, due dates, and late fees",
    icon: CurrencyInr,
  },
  {
    href: "/admin/publish-results",
    title: "Publish results",
    description: "Make faculty drafts visible to students",
    icon: Exam,
  },
  {
    href: "/admin/announcements",
    title: "Announcements",
    description: "Reach students by program or batch",
    icon: Megaphone,
  },
  {
    href: "/admin/bulk-import",
    title: "Bulk import",
    description: "CSV upload for people and results",
    icon: UploadSimple,
  },
  {
    href: "/admin/branding",
    title: "Branding",
    description: "Logo and theme for your institute",
    icon: Palette,
  },
  {
    href: "/admin/audit",
    title: "Audit log",
    description: "Who changed what, and when",
    icon: Scroll,
  },
];

export default function AdminDashboardPage() {
  return (
    <div>
      <DashboardGreeting subtitle="Configure academics, fees, and communications for your institute." />
      <section aria-labelledby="admin-quick-links">
        <h2 id="admin-quick-links" className="sr-only">
          Quick links
        </h2>
        <QuickLinkGrid links={links} />
      </section>
    </div>
  );
}
