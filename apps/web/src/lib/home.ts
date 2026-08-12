import type { Role } from "@unicore/shared";

export function homeForRole(role: Role): string {
  switch (role) {
    case "student":
      return "/student";
    case "faculty":
      return "/faculty";
    case "institute_admin":
      return "/admin";
    case "superadmin":
      return "/superadmin";
    default:
      return "/";
  }
}
