"use client";

import { ClipboardText, Exam, Megaphone, UsersThree } from "@phosphor-icons/react";
import {
  DashboardGreeting,
  QuickLinkGrid,
  type QuickLink,
} from "@/components/nav-shell";

const links: QuickLink[] = [
  {
    href: "/faculty/attendance",
    title: "Mark attendance",
    description: "Record today’s session for your courses",
    icon: ClipboardText,
  },
  {
    href: "/faculty/results",
    title: "Enter results",
    description: "Draft grades before admin publish",
    icon: Exam,
  },
  {
    href: "/faculty/roster",
    title: "Course roster",
    description: "See enrolled students per course",
    icon: UsersThree,
  },
  {
    href: "/faculty/announcements",
    title: "Announcements",
    description: "Post a notice to the campus feed",
    icon: Megaphone,
  },
];

export default function FacultyDashboardPage() {
  return (
    <div>
      <DashboardGreeting subtitle="Teach, mark, and review — start with attendance or results." />
      <section aria-labelledby="faculty-quick-links">
        <h2 id="faculty-quick-links" className="sr-only">
          Quick links
        </h2>
        <QuickLinkGrid links={links} />
      </section>
    </div>
  );
}
