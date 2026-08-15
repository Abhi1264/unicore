package handlers

import (
	"time"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AttendanceHandler struct {
	svc  *services.AttendanceService
	pool *db.Pool
}

func NewAttendanceHandler(svc *services.AttendanceService, pool *db.Pool) *AttendanceHandler {
	return &AttendanceHandler{svc: svc, pool: pool}
}

func (h *AttendanceHandler) Mark(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		StudentID   uuid.UUID `json:"student_id"`
		CourseID    uuid.UUID `json:"course_id"`
		SessionDate string    `json:"session_date"`
		Status      string    `json:"status"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.StudentID == uuid.Nil || body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "student_id and course_id are required")
	}
	if err := assertStudentInTenant(c.Context(), h.pool, tenantID, body.StudentID); err != nil {
		return mapSvcError(c, err)
	}
	if err := assertCanTeach(c, h.pool, tenantID, body.CourseID, ""); err != nil {
		return err
	}
	sessionDate, err := time.Parse("2006-01-02", body.SessionDate)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "session_date must be YYYY-MM-DD")
	}
	rec, err := h.svc.MarkAttendance(c.Context(), tenantID, services.MarkAttendanceInput{
		StudentID: body.StudentID, CourseID: body.CourseID,
		SessionDate: sessionDate, Status: body.Status, MarkedBy: claims.UserID,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, rec)
}

func (h *AttendanceHandler) MarkSession(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		CourseID    uuid.UUID              `json:"course_id"`
		Semester    string                 `json:"semester"`
		SessionDate string                 `json:"session_date"`
		Marks       []services.SessionMark `json:"marks"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	if err := assertCanTeach(c, h.pool, tenantID, body.CourseID, body.Semester); err != nil {
		return err
	}
	sessionDate, err := time.Parse("2006-01-02", body.SessionDate)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "session_date must be YYYY-MM-DD")
	}
	for _, m := range body.Marks {
		if err := assertStudentInTenant(c.Context(), h.pool, tenantID, m.StudentID); err != nil {
			return mapSvcError(c, err)
		}
	}
	recs, err := h.svc.MarkSession(c.Context(), tenantID, services.MarkSessionInput{
		CourseID: body.CourseID, SessionDate: sessionDate, Marks: body.Marks, MarkedBy: claims.UserID,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"attendance": recs})
}

func (h *AttendanceHandler) Session(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	courseID, err := uuid.Parse(c.Query("course_id"))
	if err != nil || courseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	sessionDate, err := time.Parse("2006-01-02", c.Query("session_date"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "session_date must be YYYY-MM-DD")
	}
	rows, err := h.svc.ListCourseSession(c.Context(), tenantID, courseID, sessionDate)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"marks": rows})
}

func (h *AttendanceHandler) Summary(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	studentID, err := resolveStudentScope(c, h.pool, tenantID, claims)
	if err != nil {
		return err
	}
	sum, err := h.svc.StudentAttendanceSummary(c.Context(), tenantID, studentID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, sum)
}
