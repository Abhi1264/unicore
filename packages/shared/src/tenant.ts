import { z } from "zod";

// These must match the Postgres enums in 000001_init_schema.up.sql. A value
// that is valid here but not there fails deep in the driver as an opaque 500.
export const TenantStatusSchema = z.enum([
  "pending_approval",
  "active",
  "suspended",
  "rejected",
]);
export type TenantStatus = z.infer<typeof TenantStatusSchema>;

export const GradingSystemSchema = z.enum(["cgpa", "percentage", "letter"]);
export type GradingSystem = z.infer<typeof GradingSystemSchema>;

export const AcademicCalendarTypeSchema = z.enum(["semester", "annual"]);
export type AcademicCalendarType = z.infer<typeof AcademicCalendarTypeSchema>;

const HEX_COLOR = /^#[0-9A-Fa-f]{6}$/;

/**
 * Rejects anything that is not an absolute http(s) URL.
 *
 * `z.string().url()` is not enough on its own: it delegates to the URL
 * constructor, which happily accepts `javascript:alert(1)` and `data:` URIs.
 * Those become script execution as soon as the value reaches an `href` or an
 * image handler. The API enforces the same rule server-side; this copy exists
 * to give the admin immediate feedback in the form.
 */
const httpUrl = z
  .string()
  .max(2048)
  .refine((value) => {
    try {
      const { protocol } = new URL(value);
      return protocol === "http:" || protocol === "https:";
    } catch {
      return false;
    }
  }, "Must be an absolute http(s) URL");

export const BrandingConfigSchema = z.object({
  primary_color: z
    .string()
    .regex(HEX_COLOR, "Must be a hex color like #1A73E8")
    .optional(),
  secondary_color: z.string().regex(HEX_COLOR).optional(),
  logo_url: httpUrl.nullable().optional(),
  favicon_url: httpUrl.nullable().optional(),
  institute_display_name: z.string().min(1).max(200).optional(),
});
export type BrandingConfig = z.infer<typeof BrandingConfigSchema>;

export const TenantPublicSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(200),
  subdomain: z.string().min(2).max(63),
  status: TenantStatusSchema,
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
});
export type TenantPublic = z.infer<typeof TenantPublicSchema>;

export const TenantConfigPublicSchema = z.object({
  tenant_id: z.string().uuid(),
  grading_system: GradingSystemSchema,
  branding: BrandingConfigSchema,
  academic_year_start_month: z.number().int().min(1).max(12).default(7),
  timezone: z.string().min(1).max(64),
  locale: z.string().min(2).max(16).default("en-IN"),
});
export type TenantConfigPublic = z.infer<typeof TenantConfigPublicSchema>;
