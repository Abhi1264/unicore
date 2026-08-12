"use client";

import { Buildings, ChartBar } from "@phosphor-icons/react";
import {
  DashboardGreeting,
  QuickLinkGrid,
  type QuickLink,
} from "@/components/nav-shell";

const links: QuickLink[] = [
  {
    href: "/superadmin/tenants",
    title: "Tenant queue",
    description: "Approve, reject, suspend, or reactivate institutes",
    icon: Buildings,
  },
  {
    href: "/superadmin/usage",
    title: "Usage",
    description: "Cross-tenant activity and health signals",
    icon: ChartBar,
  },
];

export default function SuperadminDashboardPage() {
  return (
    <div>
      <DashboardGreeting subtitle="Platform control — tenant lifecycle and system usage." />
      <section aria-labelledby="superadmin-quick-links">
        <h2 id="superadmin-quick-links" className="sr-only">
          Quick links
        </h2>
        <QuickLinkGrid links={links} />
      </section>
    </div>
  );
}
