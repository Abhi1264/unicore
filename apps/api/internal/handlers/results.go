package handlers

import (
	"math"

	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ResultsHandler struct {
	svc  *services.ResultsService
	pool *db.Pool
}

func NewResultsHandler(svc *services.ResultsService, pool *db.Pool) *ResultsHandler {
	return &ResultsHandler{svc: svc, pool: pool}
}

func (h *ResultsHandler) GetMine(c *fiber.Ctx) error {
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

	var semPtr *string
	if s := c.Query("semester"); s != "" {
		if err := requireSemester(s); err != nil {
			return err
		}
		semPtr = &s
	}
	res, err := h.svc.GetStudentResults(c.Context(), tenantID, studentID, semPtr)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, res)
}

func (h *ResultsHandler) Enter(c *fiber.Ctx) error {
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
		Semester    string    `json:"semester"`
		Grade       string    `json:"grade"`
		GradePoints *float64  `json:"grade_points"`
		Marks       *float64  `json:"marks"`
		Status      string    `json:"status"`
	}
	if err := parseBody(c, &body); err != nil {
		return err
	}
	if body.StudentID == uuid.Nil || body.CourseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "student_id and course_id are required")
	}
	if err := requireSemester(body.Semester); err != nil {
		return err
	}
	if err := requireText("grade", body.Grade, 8); err != nil {
		return err
	}
	if err := boundedOptional("grade_points", body.GradePoints, 0, 10); err != nil {
		return err
	}
	if err := boundedOptional("marks", body.Marks, 0, 1000); err != nil {
		return err
	}

	row, err := h.svc.EnterResult(c.Context(), tenantID, services.EnterResultInput{
		StudentID:   body.StudentID,
		CourseID:    body.CourseID,
		Semester:    body.Semester,
		Grade:       body.Grade,
		GradePoints: body.GradePoints,
		Marks:       body.Marks,
		Status:      body.Status,
		EnteredBy:   claims.UserID,
	})
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusCreated, row)
}

func (h *ResultsHandler) EnterBatch(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
	if err != nil {
		return err
	}
	var body struct {
		CourseID uuid.UUID `json:"course_id"`
		Semester string    `json:"semester"`
		Status   string    `json:"status"`
		Rows     []struct {
			StudentID   uuid.UUID `json:"student_id"`
			Grade       string    `json:"grade"`
			GradePoints *float64  `json:"grade_points"`
			Marks       *float64  `json:"marks"`
		} `json:"rows"`
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
	inputs := make([]services.EnterResultInput, 0, len(body.Rows))
	for _, row := range body.Rows {
		if row.StudentID == uuid.Nil {
			return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "student_id is required")
		}
		if err := requireText("grade", row.Grade, 8); err != nil {
			return err
		}
		if err := boundedOptional("grade_points", row.GradePoints, 0, 10); err != nil {
			return err
		}
		if err := boundedOptional("marks", row.Marks, 0, 1000); err != nil {
			return err
		}
		if err := assertStudentInTenant(c.Context(), h.pool, tenantID, row.StudentID); err != nil {
			return mapSvcError(c, err)
		}
		inputs = append(inputs, services.EnterResultInput{
			StudentID:   row.StudentID,
			CourseID:    body.CourseID,
			Semester:    body.Semester,
			Grade:       row.Grade,
			GradePoints: row.GradePoints,
			Marks:       row.Marks,
			Status:      body.Status,
			EnteredBy:   claims.UserID,
		})
	}
	rows, err := h.svc.EnterResults(c.Context(), tenantID, inputs)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"results": rows})
}

func (h *ResultsHandler) ListCourse(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	courseID, err := uuid.Parse(c.Query("course_id"))
	if err != nil || courseID == uuid.Nil {
		return JSONError(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "course_id is required")
	}
	semester := c.Query("semester")
	if err := requireSemester(semester); err != nil {
		return err
	}
	rows, err := h.svc.ListCourseResults(c.Context(), tenantID, courseID, semester)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"results": rows})
}

func (h *ResultsHandler) Publish(c *fiber.Ctx) error {
	tenantID, err := requireTenantID(c)
	if err != nil {
		return err
	}
	claims, err := requireClaims(c)
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
	rows, err := h.svc.PublishCourseResults(c.Context(), tenantID, body.CourseID, body.Semester, claims.UserID)
	if err != nil {
		return mapSvcError(c, err)
	}
	return JSON(c, fiber.StatusOK, fiber.Map{"results": rows})
}

func boundedOptional(field string, v *float64, min, max float64) error {
	if v == nil {
		return nil
	}
	if math.IsNaN(*v) || math.IsInf(*v, 0) || *v < min || *v > max {
		return fiber.NewError(fiber.StatusBadRequest, field+" is out of range")
	}
	return nil
}
