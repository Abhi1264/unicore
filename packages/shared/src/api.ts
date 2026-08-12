import { z } from "zod";

export const ApiErrorSchema = z.object({
  error: z.string().min(1),
  code: z.string().min(1).optional(),
  request_id: z.string().min(1).optional(),
});
export type ApiError = z.infer<typeof ApiErrorSchema>;
