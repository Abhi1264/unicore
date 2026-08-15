package handlers

import (
	"fmt"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AcademicHandler struct {
	svc  *services.AcademicService
	pool *db.Pool
}

func NewAcademicHandler(svc *services.AcademicService, pool *db.Pool) *AcademicHandler {
	return &AcademicHandler{svc: svc, pool: pool}
}

func (h *AcademicHandler) ListDepartments(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	list, err := h.svc.ListDepartments(c.Context(), tenantID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"departments": list})
}

func (h *AcademicHandler) CreateDepartment(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if err := requireText("code", body.Code, 32); err != nil {
		return err
	}
	if err := requireText("name", body.Name, 200); err != nil {
		return err
	}
	d, err := h.svc.CreateDepartment(c.Context(), tenantID, body.Code, body.Name)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, d)
}

func (h *AcademicHandler) ListCourses(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	list, err := h.svc.ListCourses(c.Context(), tenantID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"courses": list})
}

func (h *AcademicHandler) CreateCourse(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		Code         string     `json:"code"`
		Name         string     `json:"name"`
		Credits      float64    `json:"credits"`
		DepartmentID *uuid.UUID `json:"department_id"`
		SeatCap      int32      `json:"seat_cap"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if err := requireText("code", body.Code, 32); err != nil {
		return err
	}
	if err := requireText("name", body.Name, 200); err != nil {
		return err
	}
	if body.Credits < 0 || body.Credits > 100 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "credits must be between 0 and 100")
	}
	if body.SeatCap < 0 || body.SeatCap > 100000 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "seat_cap must be between 0 and 100000")
	}
	course, err := h.svc.CreateCourse(c.Context(), tenantID, services.CreateCourseInput{
		Code: body.Code, Name: body.Name, Credits: body.Credits,
		DepartmentID: body.DepartmentID, SeatCap: body.SeatCap,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, course)
}

func (h *AcademicHandler) Enroll(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	key, err := requireIdempotencyKey(c)
	if err != nil {
		return err
	}
	var body struct {
		CourseID uuid.UUID `json:"course_id"`
		Semester string    `json:"semester"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	if err := requireSemester(body.Semester); err != nil {
		return err
	}
	// The route is student-only and the enrolling student is always the caller,
	// so there is no request field that could name someone else to enroll.
	sid, err := studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}
	enr, err := h.svc.EnrollStudent(c.Context(), tenantID, services.EnrollStudentInput{
		StudentID: sid, CourseID: body.CourseID, Semester: body.Semester, IdempotencyKey: key,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, enr)
}

func (h *AcademicHandler) Drop(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		CourseID  uuid.UUID `json:"course_id"`
		Semester  string    `json:"semester"`
		StudentID uuid.UUID `json:"student_id"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	if err := requireSemester(body.Semester); err != nil {
		return err
	}

	// A student can only ever drop their own enrollment; student_id in the body
	// is ignored for them so it cannot be used to drop a classmate's course.
	var sid uuid.UUID
	if claims.Role == auth.RoleStudent {
		sid, err = studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
		if err != nil {
			return mapSvcError(c, err)
		}
	} else {
		if body.StudentID == uuid.Nil {
			return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "student_id is required")
		}
		if err := assertStudentInTenant(c.Context(), h.pool, tenantID, body.StudentID); err != nil {
			return mapSvcError(c, err)
		}
		sid = body.StudentID
	}

	enr, err := h.svc.DropEnrollment(c.Context(), tenantID, sid, body.CourseID, body.Semester)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, enr)
}

func (h *AcademicHandler) MyEnrollments(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	sid, err := studentIDForUser(c.Context(), h.pool, tenantID, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}
	list, err := h.svc.ListMyEnrollments(c.Context(), tenantID, sid)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"enrollments": list})
}

func (h *AcademicHandler) Roster(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	courseID, err := parseUUIDParam(c, "courseId")
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid course id")
	}
	semester := c.Query("semester", "")
	if err := requireSemester(semester); err != nil {
		return err
	}
	list, err := h.svc.ListRoster(c.Context(), tenantID, courseID, semester)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"roster": list})
}

func (h *AcademicHandler) ListTimetable(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	sem := c.Query("semester", "")
	if err := requireSemester(sem); err != nil {
		return err
	}
	list, err := h.svc.ListTimetable(c.Context(), tenantID, sem)
	if err != nil {
		return mapSvcError(c, err)
	}
	slots := make([]fiber.Map, 0, len(list))
	for _, s := range list {
		room := ""
		if s.Room.Valid {
			room = s.Room.String
		}
		slots = append(slots, fiber.Map{
			"id":          s.ID,
			"course_id":   s.CourseID,
			"course_code": s.CourseCode,
			"course_name": s.CourseName,
			"semester":    s.Semester,
			"day_of_week": s.DayOfWeek,
			"start_time":  formatPGTime(s.StartTime),
			"end_time":    formatPGTime(s.EndTime),
			"room":        room,
		})
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"slots": slots, "entries": slots})
}

func (h *AcademicHandler) CreateTimetable(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		CourseID  uuid.UUID `json:"course_id"`
		Semester  string    `json:"semester"`
		DayOfWeek int32     `json:"day_of_week"`
		StartHour int       `json:"start_hour"`
		StartMin  int       `json:"start_min"`
		EndHour   int       `json:"end_hour"`
		EndMin    int       `json:"end_min"`
		Room      string    `json:"room"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	if err := requireSemester(body.Semester); err != nil {
		return err
	}
	if body.DayOfWeek < 0 || body.DayOfWeek > 6 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "day_of_week must be 0-6")
	}
	if !validClock(body.StartHour, body.StartMin) || !validClock(body.EndHour, body.EndMin) {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "invalid start or end time")
	}
	if body.EndHour*60+body.EndMin <= body.StartHour*60+body.StartMin {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "end time must be after start time")
	}
	if len(body.Room) > 64 {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "room is too long")
	}
	slot, err := h.svc.CreateTimetableSlot(c.Context(), tenantID, services.CreateTimetableSlotInput{
		CourseID: body.CourseID, Semester: body.Semester, DayOfWeek: body.DayOfWeek,
		StartHour: body.StartHour, StartMin: body.StartMin, EndHour: body.EndHour, EndMin: body.EndMin,
		Room: body.Room,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, slot)
}

func validClock(hour, minute int) bool {
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func formatPGTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	sec := t.Microseconds / 1_000_000
	h := sec / 3600
	m := (sec % 3600) / 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func (h *AcademicHandler) CreateRegWindow(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	var body struct {
		Name     string    `json:"name"`
		Semester string    `json:"semester"`
		OpensAt  time.Time `json:"opens_at"`
		ClosesAt time.Time `json:"closes_at"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if err := requireText("name", body.Name, 200); err != nil {
		return err
	}
	if err := requireSemester(body.Semester); err != nil {
		return err
	}
	if body.OpensAt.IsZero() || body.ClosesAt.IsZero() || !body.ClosesAt.After(body.OpensAt) {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "closes_at must be after opens_at")
	}
	w, err := h.svc.CreateRegistrationWindow(c.Context(), tenantID, services.CreateRegistrationWindowInput{
		Name: body.Name, Semester: body.Semester, OpensAt: body.OpensAt, ClosesAt: body.ClosesAt,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, w)
}

func (h *AcademicHandler) GetOpenRegWindow(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	semester := c.Query("semester", "")
	if err := requireSemester(semester); err != nil {
		return err
	}
	w, err := h.svc.GetOpenRegistrationWindow(c.Context(), tenantID, semester)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, w)
}
