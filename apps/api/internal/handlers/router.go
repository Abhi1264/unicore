package handlers

import (
	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/config"
	"github.com/Abhi1264/unicore/api/internal/metrics"
	"github.com/Abhi1264/unicore/api/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// Register wires all HTTP routes onto app.
func Register(app *fiber.App, cfg *config.Config, deps Deps) {
	authH := NewAuthHandler(deps.Auth, deps.Pool, deps.Tokens, cfg.IsProduction())
	tenantsH := NewTenantsHandler(deps.Admin, deps.Pool)
	resultsH := NewResultsHandler(deps.Results, deps.Pool)
	academicH := NewAcademicHandler(deps.Academic, deps.Pool)
	feesH := NewFeesHandler(deps.Fees, deps.Pool, cfg.PaymentWebhookSecret)
	attendanceH := NewAttendanceHandler(deps.Attendance, deps.Pool)
	annH := NewAnnouncementsHandler(deps.Announcements, deps.Hub)
	docsH := NewDocumentsHandler(deps.Documents, deps.Pool, deps.StoragePath)
	adminH := NewAdminHandler(deps.Admin, deps.StoragePath, cfg.MaxUploadBytes)

	authenticated := middleware.Authenticate(deps.Tokens)
	// Per-user budgets need claims; mount after Authenticate.
	userThrottle := middleware.RateLimitUser(cfg, deps.Redis)
	writeThrottle := middleware.RateLimitWrite(cfg, deps.Redis)
	authThrottle := middleware.RateLimitAuth(cfg, deps.Redis, deps.Log)

	app.Get("/healthz", Healthz)
	app.Get("/readyz", Readyz(deps.Pool))
	app.Get("/metrics", MetricsGuard(cfg.MetricsToken, cfg.AppEnv == "development"), metrics.Handler())

	api := app.Group("/api/v1")
	api.Use(middleware.RequireTrustedOrigin(cfg))

	api.Post("/auth/register-tenant", authThrottle, authH.RegisterTenant)
	api.Post("/auth/login", authThrottle, authH.Login)
	api.Post("/auth/refresh", authThrottle, authH.Refresh)
	// Logout is unauthenticated (refresh token body); keep the tight auth budget.
	api.Post("/auth/logout", authThrottle, authH.Logout)
	api.Get("/auth/me", authenticated, userThrottle, authH.Me)

	api.Get("/tenants/current", authenticated, userThrottle, tenantsH.GetCurrent)
	api.Patch("/tenants/current/branding",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		tenantsH.UpdateBranding,
	)

	// Platform control plane — never served from institute hosts.
	sa := api.Group("/admin/tenants",
		middleware.RequirePlatformHost(),
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleSuperadmin),
	)
	sa.Get("/", tenantsH.List)
	sa.Post("/:id/approve", tenantsH.Approve)
	sa.Post("/:id/reject", tenantsH.Reject)
	sa.Post("/:id/suspend", tenantsH.Suspend)
	sa.Post("/:id/reactivate", tenantsH.Reactivate)

	api.Get("/results/me",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleFaculty, auth.RoleInstituteAdmin),
		resultsH.GetMine,
	)
	api.Post("/results",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		resultsH.Enter,
	)
	api.Post("/results/batch",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		resultsH.EnterBatch,
	)
	api.Get("/results/course",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		resultsH.ListCourse,
	)
	api.Post("/results/publish",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		resultsH.Publish,
	)

	api.Get("/departments", authenticated, userThrottle, academicH.ListDepartments)
	api.Post("/departments",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		academicH.CreateDepartment,
	)
	api.Get("/courses", authenticated, userThrottle, academicH.ListCourses)
	api.Post("/courses",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		academicH.CreateCourse,
	)
	api.Post("/enrollments",
		authenticated,
		userThrottle,
		writeThrottle,
		middleware.RequireRoles(auth.RoleStudent),
		academicH.Enroll,
	)
	api.Get("/enrollments/me",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent),
		academicH.MyEnrollments,
	)
	api.Post("/enrollments/drop",
		authenticated,
		userThrottle,
		writeThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleInstituteAdmin),
		academicH.Drop,
	)
	api.Get("/courses/:courseId/roster",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		academicH.Roster,
	)
	api.Get("/timetable", authenticated, userThrottle, academicH.ListTimetable)
	api.Post("/timetable",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		academicH.CreateTimetable,
	)
	api.Get("/registration-windows/open", authenticated, userThrottle, academicH.GetOpenRegWindow)
	api.Post("/registration-windows",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		academicH.CreateRegWindow,
	)

	api.Get("/fees/heads", authenticated, userThrottle, feesH.ListHeads)
	api.Post("/fees/heads",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		feesH.CreateHead,
	)
	api.Get("/fees/dues",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleInstituteAdmin),
		feesH.ListDues,
	)
	api.Post("/fees/pay",
		authenticated,
		userThrottle,
		writeThrottle,
		middleware.RequireRoles(auth.RoleStudent),
		feesH.Pay,
	)
	// Provider callback: HMAC over body, not bearer auth.
	api.Post("/fees/confirm", feesH.ConfirmWebhook)

	api.Post("/attendance",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		attendanceH.Mark,
	)
	api.Post("/attendance/session",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		attendanceH.MarkSession,
	)
	api.Get("/attendance/session",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleFaculty, auth.RoleInstituteAdmin),
		attendanceH.Session,
	)
	api.Get("/attendance/summary",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleFaculty, auth.RoleInstituteAdmin),
		attendanceH.Summary,
	)

	api.Post("/announcements",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin, auth.RoleFaculty),
		annH.Create,
	)
	api.Get("/announcements", authenticated, userThrottle, annH.List)
	api.Get("/announcements/stream", authenticated, userThrottle, annH.Stream)

	api.Post("/documents",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent),
		docsH.Request,
	)
	api.Get("/documents",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleInstituteAdmin),
		docsH.List,
	)
	api.Get("/documents/:id/download",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleStudent, auth.RoleInstituteAdmin),
		docsH.Download,
	)

	api.Post("/admin/bulk-import",
		authenticated,
		userThrottle,
		writeThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		adminH.BulkImport,
	)
	api.Get("/admin/audit-logs",
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleInstituteAdmin),
		adminH.AuditLogs,
	)
	api.Get("/admin/usage",
		middleware.RequirePlatformHost(),
		authenticated,
		userThrottle,
		middleware.RequireRoles(auth.RoleSuperadmin),
		adminH.Usage,
	)
}
