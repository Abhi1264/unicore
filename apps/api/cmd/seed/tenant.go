package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type seedStats struct {
	Students, Faculty, Courses int
	Enrollments, Results       int
	Attendance, Payments       int
	Announcements              int
}

func ensureTenant(ctx context.Context, q *sqlcdb.Queries, slug, name, status string) (sqlcdb.Tenant, error) {
	t, err := q.GetTenantBySlug(ctx, slug)
	if err == nil {
		if t.Status != status {
			return q.UpdateTenantStatus(ctx, sqlcdb.UpdateTenantStatusParams{ID: t.ID, Status: status})
		}
		return t, nil
	}
	t, err = q.CreateTenant(ctx, sqlcdb.CreateTenantParams{Slug: slug, Name: name})
	if err != nil {
		return t, err
	}
	return q.UpdateTenantStatus(ctx, sqlcdb.UpdateTenantStatusParams{ID: t.ID, Status: status})
}

func seedSuperadmin(ctx context.Context, pool *db.Pool, tenantID uuid.UUID) error {
	return pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		_, err := q.GetUserByEmail(ctx, sqlcdb.GetUserByEmailParams{
			TenantID: tenantID, Email: "superadmin@unicore.local",
		})
		if err == nil {
			return nil
		}
		hash, err := auth.HashPassword(demoPassword("superadmin"))
		if err != nil {
			return err
		}
		_, err = q.CreateUser(ctx, sqlcdb.CreateUserParams{
			TenantID: tenantID, Email: "superadmin@unicore.local", PasswordHash: hash,
			Role: "superadmin", FullName: "Platform Superadmin",
		})
		return err
	})
}

func seedTenant(ctx context.Context, pool *db.Pool, ten sqlcdb.Tenant, cfg seedConfig) (seedStats, error) {
	var stats seedStats
	var (
		admin          sqlcdb.User
		primaryFaculty sqlcdb.User
		deptByCode     map[string]sqlcdb.Department
		courses        []sqlcdb.Course
		feeHeads       []sqlcdb.FeeHead
		students       []sqlcdb.Student
	)

	// Phases use separate transactions so a conflict in one domain cannot poison
	// the rest of the seed (Postgres aborts the whole tx on unique violations).
	phase := func(name string, fn func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error) error {
		if err := pool.WithTenantTx(ctx, ten.ID, fn); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}

	stuHash, err := auth.HashPassword(demoPassword("student"))
	if err != nil {
		return stats, err
	}

	if err := phase("foundation", func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error {
		if err := ensureConfig(ctx, q, ten.ID); err != nil {
			return err
		}
		var err error
		admin, err = ensureUser(ctx, q, ten.ID, "admin@"+ten.Slug+".edu", "institute_admin", ten.Name+" Admin", "admin")
		if err != nil {
			return err
		}
		deptByCode, err = ensureDepartments(ctx, q, ten.ID)
		if err != nil {
			return err
		}
		facultyUsers, err := ensureFaculty(ctx, q, ten, deptByCode, cfg.Faculty)
		if err != nil {
			return err
		}
		stats.Faculty = len(facultyUsers)
		primaryFaculty = facultyUsers[0]
		courses, err = ensureCourses(ctx, q, ten.ID, deptByCode, cfg.Students)
		if err != nil {
			return err
		}
		stats.Courses = len(courses)
		if err := ensureRegWindow(ctx, q, ten.ID); err != nil {
			return err
		}
		if err := ensureTimetable(ctx, tx, q, ten.ID, courses); err != nil {
			return err
		}
		feeHeads, err = ensureFeeHeads(ctx, q, ten.ID)
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("students", func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		if _, err := ensureNamedStudent(ctx, q, ten, deptByCode["CSE"].ID, stuHash); err != nil {
			return err
		}
		if _, err := ensureBulkStudents(ctx, q, ten, deptByCode, stuHash, cfg.Students); err != nil {
			return err
		}
		var err error
		students, err = q.ListStudents(ctx, ten.ID)
		stats.Students = len(students)
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("enrollments", func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error {
		nEnroll, nResults, err := seedEnrollmentsAndResults(ctx, tx, q, ten.ID, students, courses, primaryFaculty.ID)
		stats.Enrollments, stats.Results = nEnroll, nResults
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("attendance", func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		n, err := seedAttendance(ctx, q, ten.ID, students, courses, primaryFaculty.ID, cfg)
		stats.Attendance = n
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("payments", func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error {
		n, err := seedPayments(ctx, tx, q, ten.ID, students, feeHeads)
		stats.Payments = n
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("announcements", func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		n, err := seedAnnouncements(ctx, q, ten.ID, admin.ID)
		stats.Announcements = n
		return err
	}); err != nil {
		return stats, err
	}

	if err := phase("documents", func(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries) error {
		return seedDocuments(ctx, tx, q, ten.ID, students)
	}); err != nil {
		return stats, err
	}

	_ = phase("audit", func(ctx context.Context, _ pgx.Tx, q *sqlcdb.Queries) error {
		meta, _ := json.Marshal(map[string]int{
			"students": stats.Students, "courses": stats.Courses,
		})
		return q.InsertAuditLog(ctx, sqlcdb.InsertAuditLogParams{
			TenantID: ten.ID, ActorID: uuidToPg(admin.ID),
			Action: "seed.completed", Entity: "tenant", EntityID: uuidToPg(ten.ID),
			Metadata: meta,
		})
	})
	return stats, nil
}

// ignoreUnique rolls back to a savepoint when a unique/check conflict fires so
// the surrounding seed transaction stays usable.
func ignoreUnique(ctx context.Context, tx pgx.Tx, fn func() error) error {
	sp := "sp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
		return err
	}
	if err := fn(); err != nil {
		_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp)
		if isBenignConflict(err) {
			_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT "+sp)
			return nil
		}
		_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT "+sp)
		return err
	}
	_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT "+sp)
	return nil
}

func isBenignConflict(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "sqlstate 23505") ||
		strings.Contains(msg, "uniq_fee_payment")
}

func ensureConfig(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID) error {
	if _, err := q.GetTenantConfig(ctx, tenantID); err == nil {
		return nil
	}
	branding, _ := json.Marshal(map[string]any{
		"primary_color":   "#0d3d3a",
		"secondary_color": "#1a5c57",
	})
	scale, _ := json.Marshal(map[string]any{
		"A+": 10, "A": 9, "B+": 8, "B": 7, "C": 6, "D": 5, "F": 0,
	})
	_, err := q.CreateTenantConfig(ctx, sqlcdb.CreateTenantConfigParams{
		TenantID: tenantID, GradingSystem: "cgpa", AcademicCalendarType: "semester",
		Branding: branding, GradingScale: scale,
		AttendanceThresholdPct: services.NumericFromFloat(75),
	})
	return err
}

func ensureDepartments(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID) (map[string]sqlcdb.Department, error) {
	out := map[string]sqlcdb.Department{}
	existing, _ := q.ListDepartments(ctx, tenantID)
	for _, d := range existing {
		out[d.Code] = d
	}
	for _, spec := range departments {
		if _, ok := out[spec.Code]; ok {
			continue
		}
		d, err := q.CreateDepartment(ctx, sqlcdb.CreateDepartmentParams{
			TenantID: tenantID, Code: spec.Code, Name: spec.Name,
		})
		if err != nil {
			return nil, err
		}
		out[spec.Code] = d
	}
	return out, nil
}

func ensureFaculty(ctx context.Context, q *sqlcdb.Queries, ten sqlcdb.Tenant, depts map[string]sqlcdb.Department, n int) ([]sqlcdb.User, error) {
	if n < 1 {
		n = 1
	}
	deptCodes := []string{"CSE", "ECE", "ME", "MATH"}
	users := make([]sqlcdb.User, 0, n)

	primary, err := ensureUser(ctx, q, ten.ID, "faculty@"+ten.Slug+".edu", "faculty", "Demo Faculty", "faculty")
	if err != nil {
		return nil, err
	}
	if _, err := q.GetFacultyByUserID(ctx, sqlcdb.GetFacultyByUserIDParams{TenantID: ten.ID, UserID: primary.ID}); err != nil {
		if _, err = q.CreateFaculty(ctx, sqlcdb.CreateFacultyParams{
			TenantID: ten.ID, UserID: primary.ID,
			DepartmentID: uuidToPg(depts["CSE"].ID), EmployeeID: textPtr("F001"),
		}); err != nil {
			return nil, err
		}
	}
	users = append(users, primary)

	for i := 1; i < n; i++ {
		code := deptCodes[i%len(deptCodes)]
		email := fmt.Sprintf("faculty%02d@%s.edu", i, ten.Slug)
		u, err := ensureUser(ctx, q, ten.ID, email, "faculty",
			fmt.Sprintf("Faculty %s %d", code, i), "faculty")
		if err != nil {
			return nil, err
		}
		if _, err := q.GetFacultyByUserID(ctx, sqlcdb.GetFacultyByUserIDParams{TenantID: ten.ID, UserID: u.ID}); err != nil {
			if _, err = q.CreateFaculty(ctx, sqlcdb.CreateFacultyParams{
				TenantID: ten.ID, UserID: u.ID,
				DepartmentID: uuidToPg(depts[code].ID),
				EmployeeID:   textPtr(fmt.Sprintf("F%03d", i+1)),
			}); err != nil {
				return nil, err
			}
		}
		users = append(users, u)
	}
	return users, nil
}

func ensureCourses(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID, depts map[string]sqlcdb.Department, students int) ([]sqlcdb.Course, error) {
	seatCap := students + 40
	if seatCap < 60 {
		seatCap = 60
	}
	existing, _ := q.ListCourses(ctx, tenantID)
	byCode := map[string]sqlcdb.Course{}
	for _, c := range existing {
		byCode[c.Code] = c
	}
	out := make([]sqlcdb.Course, 0, len(courseCatalog))
	for _, spec := range courseCatalog {
		if c, ok := byCode[spec.Code]; ok {
			out = append(out, c)
			continue
		}
		dept := depts[spec.Dept]
		c, err := q.CreateCourse(ctx, sqlcdb.CreateCourseParams{
			TenantID: tenantID, Code: spec.Code, Name: spec.Name,
			Credits:      services.NumericFromFloat(spec.Credits),
			DepartmentID: uuidToPg(dept.ID), SeatCap: int32(seatCap),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func ensureRegWindow(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID) error {
	_, err := q.GetOpenRegistrationWindow(ctx, sqlcdb.GetOpenRegistrationWindowParams{
		TenantID: tenantID, Semester: "2026S1",
	})
	if err == nil {
		return nil
	}
	_, err = q.CreateRegistrationWindow(ctx, sqlcdb.CreateRegistrationWindowParams{
		TenantID: tenantID, Name: "Spring 2026", Semester: "2026S1",
		OpensAt: time.Now().Add(-48 * time.Hour), ClosesAt: time.Now().Add(45 * 24 * time.Hour),
	})
	return err
}

func ensureTimetable(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries, tenantID uuid.UUID, courses []sqlcdb.Course) error {
	existing, err := q.ListTimetableForSemester(ctx, sqlcdb.ListTimetableForSemesterParams{
		TenantID: tenantID, Semester: "2026S1",
	})
	if err == nil && len(existing) > 0 {
		return nil
	}
	rooms := []string{"LH-1", "LH-2", "LH-3", "Lab-A", "Lab-B"}
	for i, c := range courses {
		day := int32((i % 5) + 1)
		hour := 9 + (i % 6)
		err := ignoreUnique(ctx, tx, func() error {
			_, e := q.CreateTimetableSlot(ctx, sqlcdb.CreateTimetableSlotParams{
				TenantID: tenantID, CourseID: c.ID, Semester: "2026S1",
				DayOfWeek: day, StartTime: clock(hour, 0), EndTime: clock(hour+1, 0),
				Room: textPtr(rooms[i%len(rooms)]),
			})
			return e
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureFeeHeads(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID) ([]sqlcdb.FeeHead, error) {
	existing, err := q.ListFeeHeads(ctx, tenantID)
	if err == nil && len(existing) >= len(feeCatalog) {
		return existing, nil
	}
	byName := map[string]sqlcdb.FeeHead{}
	for _, fh := range existing {
		byName[fh.Name] = fh
	}
	out := make([]sqlcdb.FeeHead, 0, len(feeCatalog))
	for _, spec := range feeCatalog {
		if fh, ok := byName[spec.Name]; ok {
			out = append(out, fh)
			continue
		}
		due := time.Now().Add(time.Duration(spec.DueDays) * 24 * time.Hour)
		fh, err := q.CreateFeeHead(ctx, sqlcdb.CreateFeeHeadParams{
			TenantID: tenantID, Name: spec.Name,
			Amount:             services.NumericFromFloat(spec.Amount),
			DueDate:            pgtype.Date{Time: due.UTC().Truncate(24 * time.Hour), Valid: true},
			LateFeeAmount:      services.NumericFromFloat(spec.LateFee),
			ApplicablePrograms: []string{"BTech", "MTech"},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, fh)
	}
	return out, nil
}

func ensureNamedStudent(ctx context.Context, q *sqlcdb.Queries, ten sqlcdb.Tenant, deptID uuid.UUID, hash string) (sqlcdb.Student, error) {
	email := "student@" + ten.Slug + ".edu"
	if u, err := q.GetUserByEmail(ctx, sqlcdb.GetUserByEmailParams{TenantID: ten.ID, Email: email}); err == nil {
		return q.GetStudentByUserID(ctx, sqlcdb.GetStudentByUserIDParams{TenantID: ten.ID, UserID: u.ID})
	}
	u, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
		TenantID: ten.ID, Email: email, PasswordHash: hash,
		Role: "student", FullName: "Demo Student",
	})
	if err != nil {
		return sqlcdb.Student{}, err
	}
	return q.CreateStudent(ctx, sqlcdb.CreateStudentParams{
		TenantID: ten.ID, UserID: u.ID, RollNumber: "BTECH24001",
		Program: "BTech", BatchYear: 2024, DepartmentID: uuidToPg(deptID),
	})
}

func ensureBulkStudents(ctx context.Context, q *sqlcdb.Queries, ten sqlcdb.Tenant, depts map[string]sqlcdb.Department, hash string, n int) (int, error) {
	deptCodes := []string{"CSE", "ECE", "ME", "MATH"}
	programs := []string{"BTech", "BTech", "BTech", "MTech"}
	created := 0
	for i := 0; i < n; i++ {
		email := fmt.Sprintf("s%04d@%s.edu", i, ten.Slug)
		if _, err := q.GetUserByEmail(ctx, sqlcdb.GetUserByEmailParams{TenantID: ten.ID, Email: email}); err == nil {
			continue
		}
		code := deptCodes[i%len(deptCodes)]
		name := firstNames[i%len(firstNames)] + " " + lastNames[(i*3)%len(lastNames)]
		u, err := q.CreateUser(ctx, sqlcdb.CreateUserParams{
			TenantID: ten.ID, Email: email, PasswordHash: hash,
			Role: "student", FullName: name,
		})
		if err != nil {
			return created, err
		}
		batch := 2022 + (i % 4)
		_, err = q.CreateStudent(ctx, sqlcdb.CreateStudentParams{
			TenantID: ten.ID, UserID: u.ID,
			RollNumber:   fmt.Sprintf("R%04d", i),
			Program:      programs[i%len(programs)],
			BatchYear:    int32(batch),
			DepartmentID: uuidToPg(depts[code].ID),
		})
		if err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func seedEnrollmentsAndResults(
	ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries, tenantID uuid.UUID,
	students []sqlcdb.Student, courses []sqlcdb.Course, facultyID uuid.UUID,
) (enrollments, results int, err error) {
	if len(courses) == 0 || len(students) == 0 {
		return 0, 0, nil
	}
	current := "2026S1"
	prior := "2025S2"
	grades := []struct {
		Letter string
		Points float64
		Marks  float64
	}{
		{"A+", 10, 92}, {"A", 9, 85}, {"B+", 8, 78}, {"B", 7, 71}, {"C", 6, 62}, {"D", 5, 52},
	}

	for si, st := range students {
		nCurrent := 4 + (si % 3)
		for k := 0; k < nCurrent && k < len(courses); k++ {
			c := courses[(si+k*3)%len(courses)]
			key := fmt.Sprintf("seed-enr-%s-%s-%s", st.ID, c.ID, current)
			created := false
			err := ignoreUnique(ctx, tx, func() error {
				if _, e := q.GetEnrollmentByIdempotency(ctx, sqlcdb.GetEnrollmentByIdempotencyParams{
					TenantID: tenantID, IdempotencyKey: textPtr(key),
				}); e == nil {
					return nil
				}
				_, e := q.CreateEnrollment(ctx, sqlcdb.CreateEnrollmentParams{
					TenantID: tenantID, StudentID: st.ID, CourseID: c.ID,
					Semester: current, IdempotencyKey: textPtr(key),
				})
				if e == nil {
					created = true
				}
				return e
			})
			if err != nil {
				return enrollments, results, err
			}
			if created {
				enrollments++
			}
		}

		nPrior := 3 + (si % 2)
		for k := 0; k < nPrior && k < len(courses); k++ {
			c := courses[(si+k*2)%len(courses)]
			g := grades[(si+k)%len(grades)]
			_, e := q.UpsertResult(ctx, sqlcdb.UpsertResultParams{
				TenantID: tenantID, StudentID: st.ID, CourseID: c.ID, Semester: prior,
				Grade: g.Letter, GradePoints: services.NumericFromFloat(g.Points),
				Marks: services.NumericFromFloat(g.Marks), SubmissionStatus: "published",
				EnteredBy: uuidToPg(facultyID),
			})
			if e != nil {
				return enrollments, results, e
			}
			results++
		}

		if si%7 == 0 {
			c := courses[si%len(courses)]
			_, _ = q.UpsertResult(ctx, sqlcdb.UpsertResultParams{
				TenantID: tenantID, StudentID: st.ID, CourseID: c.ID, Semester: current,
				Grade: "B", GradePoints: services.NumericFromFloat(7),
				Marks: services.NumericFromFloat(70), SubmissionStatus: "draft",
				EnteredBy: uuidToPg(facultyID),
			})
		}
	}

	for _, c := range courses {
		_, _ = q.PublishResultsForCourseSemester(ctx, sqlcdb.PublishResultsForCourseSemesterParams{
			TenantID: tenantID, CourseID: c.ID, Semester: prior,
		})
	}
	return enrollments, results, nil
}

func seedAttendance(
	ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID,
	students []sqlcdb.Student, courses []sqlcdb.Course, facultyID uuid.UUID, cfg seedConfig,
) (int, error) {
	if len(students) == 0 || len(courses) == 0 {
		return 0, nil
	}
	nStudents := cfg.AttendanceStudents
	if nStudents > len(students) {
		nStudents = len(students)
	}
	nSessions := cfg.AttendanceSessions
	if nSessions < 1 {
		nSessions = 1
	}
	nCourses := 3
	if nCourses > len(courses) {
		nCourses = len(courses)
	}
	statuses := []string{"present", "present", "present", "late", "absent", "excused"}
	count := 0
	for si := 0; si < nStudents; si++ {
		st := students[si]
		for ci := 0; ci < nCourses; ci++ {
			c := courses[ci]
			for s := 0; s < nSessions; s++ {
				day := time.Now().AddDate(0, 0, -(nSessions-s)*2).UTC().Truncate(24 * time.Hour)
				status := statuses[(si+ci+s)%len(statuses)]
				_, err := q.UpsertAttendance(ctx, sqlcdb.UpsertAttendanceParams{
					TenantID: tenantID, StudentID: st.ID, CourseID: c.ID,
					SessionDate: pgtype.Date{Time: day, Valid: true},
					Status:      status, MarkedBy: uuidToPg(facultyID),
				})
				if err != nil {
					return count, err
				}
				count++
			}
		}
	}
	return count, nil
}

func seedPayments(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries, tenantID uuid.UUID, students []sqlcdb.Student, heads []sqlcdb.FeeHead) (int, error) {
	if len(heads) == 0 || len(students) == 0 {
		return 0, nil
	}
	count := 0
	for i, st := range students {
		for hi, fh := range heads {
			pay := false
			switch {
			case hi == 0 && i%100 < 55:
				pay = true
			case hi > 0 && i%100 < 30:
				pay = true
			}
			if !pay {
				continue
			}
			key := fmt.Sprintf("seed-pay-%s-%s", st.ID, fh.ID)
			created := false
			err := ignoreUnique(ctx, tx, func() error {
				if _, e := q.GetFeePaymentByIdempotency(ctx, sqlcdb.GetFeePaymentByIdempotencyParams{
					TenantID: tenantID, IdempotencyKey: key,
				}); e == nil {
					return nil
				}
				amt, _ := services.FloatFromNumeric(fh.Amount)
				pmt, e := q.CreateFeePayment(ctx, sqlcdb.CreateFeePaymentParams{
					TenantID: tenantID, StudentID: st.ID, FeeHeadID: fh.ID,
					Amount: services.NumericFromFloat(amt), Status: "pending",
					IdempotencyKey: key, GatewayRef: textPtr("seed"),
				})
				if e != nil {
					return e
				}
				if _, e = q.MarkFeePaymentPaid(ctx, sqlcdb.MarkFeePaymentPaidParams{
					TenantID: tenantID, ID: pmt.ID,
					GatewayRef: textPtr("GW-SEED-" + pmt.ID.String()[:8]),
				}); e != nil {
					return e
				}
				created = true
				return nil
			})
			if err != nil {
				return count, err
			}
			if created {
				count++
			}
		}
	}
	return count, nil
}

func seedAnnouncements(ctx context.Context, q *sqlcdb.Queries, tenantID, authorID uuid.UUID) (int, error) {
	existing, err := q.ListAnnouncements(ctx, sqlcdb.ListAnnouncementsParams{TenantID: tenantID, Limit: 20})
	if err == nil && len(existing) > 0 {
		return len(existing), nil
	}
	count := 0
	for _, a := range announcementCatalog {
		filter := json.RawMessage(`{}`)
		if a.Scope == "program" {
			filter = json.RawMessage(`{"program":"BTech"}`)
		}
		_, err := q.CreateAnnouncement(ctx, sqlcdb.CreateAnnouncementParams{
			TenantID: tenantID, AuthorID: authorID,
			Title: a.Title, Body: a.Body, AudienceScope: a.Scope, AudienceFilter: filter,
		})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func seedDocuments(ctx context.Context, tx pgx.Tx, q *sqlcdb.Queries, tenantID uuid.UUID, students []sqlcdb.Student) error {
	if len(students) == 0 {
		return nil
	}
	n := 25
	if n > len(students) {
		n = len(students)
	}
	types := []string{"bonafide", "marksheet", "fee_receipt"}
	for i := 0; i < n; i++ {
		st := students[i]
		existing, _ := q.ListStudentDocuments(ctx, sqlcdb.ListStudentDocumentsParams{
			TenantID: tenantID, StudentID: st.ID,
		})
		if len(existing) > 0 {
			continue
		}
		err := ignoreUnique(ctx, tx, func() error {
			_, e := q.CreateDocument(ctx, sqlcdb.CreateDocumentParams{
				TenantID: tenantID, StudentID: st.ID, Type: types[i%len(types)],
			})
			return e
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureUser(ctx context.Context, q *sqlcdb.Queries, tenantID uuid.UUID, email, role, name, pwdRole string) (sqlcdb.User, error) {
	if u, err := q.GetUserByEmail(ctx, sqlcdb.GetUserByEmailParams{TenantID: tenantID, Email: email}); err == nil {
		return u, nil
	}
	hash, err := auth.HashPassword(demoPassword(pwdRole))
	if err != nil {
		return sqlcdb.User{}, err
	}
	return q.CreateUser(ctx, sqlcdb.CreateUserParams{
		TenantID: tenantID, Email: email, PasswordHash: hash, Role: role, FullName: name,
	})
}

func demoPassword(role string) string {
	return env("SEED_PASSWORD_PREFIX", "Unicore") + "-" + role + "-2026!"
}

func uuidToPg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func textPtr(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func clock(hour, minute int) pgtype.Time {
	us := int64(hour)*3_600_000_000 + int64(minute)*60_000_000
	return pgtype.Time{Microseconds: us, Valid: true}
}
