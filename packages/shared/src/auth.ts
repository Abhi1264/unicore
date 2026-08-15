import { z } from "zod";

export const RoleSchema = z.enum([
  "student",
  "faculty",
  "institute_admin",
  "superadmin",
]);
export type Role = z.infer<typeof RoleSchema>;

export const TokenPairSchema = z.object({
  access_token: z.string().min(1),
  refresh_token: z.string().min(1),
  token_type: z.literal("Bearer").default("Bearer"),
  expires_in: z.number().int().positive(),
});
export type TokenPair = z.infer<typeof TokenPairSchema>;

export const UserPublicSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  email: z.string().email(),
  full_name: z.string().min(1).max(200),
  role: RoleSchema,
  is_active: z.boolean(),
  created_at: z.string().datetime(),
  updated_at: z.string().datetime(),
});
export type UserPublic = z.infer<typeof UserPublicSchema>;

export const LoginRequestSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8).max(128),
});
export type LoginRequest = z.infer<typeof LoginRequestSchema>;

export const LoginResponseSchema = z.object({
  user: UserPublicSchema,
  tokens: TokenPairSchema.optional(),
});
export type LoginResponse = z.infer<typeof LoginResponseSchema>;

export const RegisterTenantRequestSchema = z.object({
  institute_name: z.string().min(2).max(200),
  subdomain: z
    .string()
    .min(2)
    .max(63)
    .regex(
      /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/,
      "Subdomain must be lowercase alphanumeric with optional hyphens",
    ),
  admin_email: z.string().email(),
  admin_full_name: z.string().min(1).max(200),
  admin_password: z.string().min(8).max(128),
  timezone: z.string().min(1).max(64).default("UTC"),
});
export type RegisterTenantRequest = z.infer<typeof RegisterTenantRequestSchema>;
