/** API response shapes aligned with the Go handlers. */

export type ResultCourseRow = {
  id: string;
  course_id: string;
  course_code: string;
  course_name: string;
  credits: number;
  semester: string;
  grade: string;
  grade_points?: number | null;
  marks?: number | null;
  submission_status: string;
  published_at?: string | null;
};

export type StudentResultsResponse = {
  student_id: string;
  semester?: string;
  grading_system: string;
  rows: ResultCourseRow[];
  cumulative_value: number;
  cumulative_display: string;
};

export type FeeHead = {
  id: string;
  name: string;
  amount: string | number;
  due_date?: string | null;
  late_fee_amount?: string | number;
  applicable_programs?: string[];
  [key: string]: unknown;
};

/** Unpaid fee head; amount_due includes any late fee. */
export type FeeDue = {
  fee_head: FeeHead;
  amount_due: number;
  late_fee_applied: boolean;
  is_overdue: boolean;
};

export type FeeDuesResponse = { dues: FeeDue[] };

export type AttendanceSummary = {
  student_id?: string;
  courses?: Array<{
    course_id: string;
    course_code?: string;
    course_name?: string;
    present: number;
    absent: number;
    late?: number;
    percentage: number;
  }>;
  overall_percentage?: number;
  [key: string]: unknown;
};

export type Course = {
  id: string;
  code: string;
  name: string;
  credits: number;
  department_id?: string | null;
  seat_cap?: number;
  [key: string]: unknown;
};

export type CoursesResponse = { courses: Course[] };

export type TimetableEntry = {
  id: string;
  course_id?: string;
  course_code?: string;
  course_name?: string;
  day_of_week?: number | string;
  start_time?: string;
  end_time?: string;
  room?: string;
  [key: string]: unknown;
};

export type TimetableResponse = {
  entries?: TimetableEntry[];
  timetable?: TimetableEntry[];
  slots?: TimetableEntry[];
};

export type Announcement = {
  id: string;
  title: string;
  body: string;
  audience_scope?: string;
  audience_filter?: {
    program?: string;
    batch_year?: number;
    course_id?: string;
  };
  created_at?: string;
  [key: string]: unknown;
};

export type AnnouncementsResponse = { announcements: Announcement[] };

export type DocumentType = "marksheet" | "bonafide" | "fee_receipt";
export type DocumentStatus = "requested" | "processing" | "ready" | "failed";

export type DocumentRequest = {
  id: string;
  type?: DocumentType;
  status?: DocumentStatus;
  created_at?: string;
  [key: string]: unknown;
};

export const DOCUMENT_TYPE_LABELS: Record<DocumentType, string> = {
  bonafide: "Bonafide certificate",
  marksheet: "Marksheet",
  fee_receipt: "Fee receipt",
};

export type DocumentsResponse = { documents: DocumentRequest[] };

export type TenantListItem = {
  id: string;
  name: string;
  slug?: string;
  subdomain?: string;
  status: string;
  created_at?: string;
  [key: string]: unknown;
};

export type TenantsQueueResponse = { tenants: TenantListItem[] };

export type UsageRow = {
  tenant_id?: string;
  subdomain?: string;
  name?: string;
  metric?: string;
  value?: number;
  [key: string]: unknown;
};

export type UsageResponse = { usage: UsageRow[] };

export type AuditLog = {
  id: string;
  action?: string;
  actor_id?: string;
  entity?: string;
  resource?: string;
  created_at?: string;
  [key: string]: unknown;
};

export type Enrollment = {
  id: string;
  course_id: string;
  course_code: string;
  course_name: string;
  credits: number;
  semester: string;
  status: string;
};

export type EnrollmentsResponse = { enrollments: Enrollment[] };

export type AuditLogsResponse = { logs: AuditLog[] };

export type RosterStudent = {
  id?: string;
  student_id: string;
  roll_number?: string;
  enrollment_number?: string;
  full_name?: string;
  email?: string;
  program?: string;
  batch_year?: number;
  semester?: string;
};

export type RosterResponse = { students?: RosterStudent[]; roster?: RosterStudent[] };

export type AttendanceMark = {
  student_id: string;
  status: "present" | "absent" | "late" | "excused";
  roll_number?: string;
  full_name?: string;
};

export type AttendanceSessionResponse = { marks: AttendanceMark[] };

export type FacultyMember = {
  id: string;
  user_id: string;
  full_name: string;
  email: string;
  employee_id?: string | { String?: string; Valid?: boolean } | null;
};

export type FacultyResponse = { faculty: FacultyMember[] };

export type CourseInstructor = {
  id: string;
  course_id: string;
  faculty_id: string;
  semester: string;
  full_name: string;
  email: string;
};

export type InstructorsResponse = { instructors: CourseInstructor[] };

export type BulkJob = {
  id: string;
  job_type: string;
  status: string;
  total_rows: number;
  success_rows: number;
  error_report?: unknown;
  created_at?: string;
  completed_at?: string | null;
};

export type BulkJobsResponse = { jobs: BulkJob[] };

export type CourseResultRow = {
  id?: string;
  student_id: string;
  roll_number?: string;
  full_name?: string;
  grade: string;
  grade_points?: number | null;
  marks?: number | null;
  submission_status: string;
};

export type CourseResultsResponse = { results: CourseResultRow[] };

export type RegistrationWindow = {
  id?: string;
  name?: string;
  semester?: string;
  opens_at?: string;
  closes_at?: string;
};
