"use client";

import {
  ClipboardText,
  CurrencyInr,
  Exam,
  Files,
  Megaphone,
  Student,
  Table,
} from "@phosphor-icons/react";
import {
  DashboardGreeting,
  QuickLinkGrid,
  type QuickLink,
} from "@/components/nav-shell";

const links: QuickLink[] = [
  {
    href: "/student/results",
    title: "Results",
    description: "Published grades and cumulative standing",
    icon: Exam,
  },
  {
    href: "/student/fees",
    title: "Fees",
    description: "View dues and pay with a receipt",
    icon: CurrencyInr,
  },
  {
    href: "/student/attendance",
    title: "Attendance",
    description: "Subject-wise presence and shortages",
    icon: ClipboardText,
  },
  {
    href: "/student/courses",
    title: "Course registration",
    description: "Add or drop within the open window",
    icon: Student,
  },
  {
    href: "/student/timetable",
    title: "Timetable",
    description: "This semester’s class schedule",
    icon: Table,
  },
  {
    href: "/student/announcements",
    title: "Announcements",
    description: "Updates from your institute",
    icon: Megaphone,
  },
  {
    href: "/student/documents",
    title: "Documents",
    description: "Request marksheets and certificates",
    icon: Files,
  },
];

export default function StudentDashboardPage() {
  return (
    <div>
      <DashboardGreeting subtitle="Your campus workspace — results, fees, courses, and more in one place." />
      <section aria-labelledby="student-quick-links">
        <h2 id="student-quick-links" className="sr-only">
          Quick links
        </h2>
        <QuickLinkGrid links={links} />
      </section>
    </div>
  );
}
