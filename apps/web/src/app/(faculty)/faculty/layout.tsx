import { NavShell } from "@/components/nav-shell";

export default function FacultyLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <NavShell role="faculty">{children}</NavShell>;
}
