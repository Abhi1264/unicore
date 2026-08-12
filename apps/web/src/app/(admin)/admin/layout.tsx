import { NavShell } from "@/components/nav-shell";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <NavShell role="institute_admin">{children}</NavShell>;
}
