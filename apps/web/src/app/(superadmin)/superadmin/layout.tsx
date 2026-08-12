import { NavShell } from "@/components/nav-shell";

export default function SuperadminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <NavShell role="superadmin">{children}</NavShell>;
}
