import { NavShell } from "@/components/nav-shell";

export default function StudentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <NavShell role="student">{children}</NavShell>;
}
