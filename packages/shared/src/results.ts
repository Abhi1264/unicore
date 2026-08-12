import { z } from "zod";

export const ResultRowSchema = z.object({
  course_code: z.string().min(1).max(32),
  course_name: z.string().min(1).max(200),
  credits: z.number().nonnegative(),
  grade: z.string().min(1).max(8),
  grade_points: z.number().nonnegative().nullable().optional(),
  marks_obtained: z.number().nonnegative().nullable().optional(),
  marks_max: z.number().positive().nullable().optional(),
  semester: z.number().int().positive(),
  academic_year: z.string().min(4).max(16),
  is_passed: z.boolean(),
});
export type ResultRow = z.infer<typeof ResultRowSchema>;

export const CumulativeSummarySchema = z.object({
  cgpa: z.number().nonnegative().nullable(),
  sgpa: z.number().nonnegative().nullable().optional(),
  total_credits_earned: z.number().nonnegative(),
  total_credits_attempted: z.number().nonnegative(),
  courses_passed: z.number().int().nonnegative(),
  courses_failed: z.number().int().nonnegative(),
  percentage: z.number().min(0).max(100).nullable().optional(),
});
export type CumulativeSummary = z.infer<typeof CumulativeSummarySchema>;

export const ResultsResponseSchema = z.object({
  student_id: z.string().uuid(),
  student_name: z.string().min(1),
  enrollment_number: z.string().min(1).max(64),
  program: z.string().min(1).max(200).optional(),
  semester: z.number().int().positive().optional(),
  academic_year: z.string().min(4).max(16).optional(),
  rows: z.array(ResultRowSchema),
  summary: CumulativeSummarySchema,
  published_at: z.string().datetime().nullable().optional(),
});
export type ResultsResponse = z.infer<typeof ResultsResponseSchema>;
