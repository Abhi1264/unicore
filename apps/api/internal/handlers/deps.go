package handlers

import (
	"log/slog"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/cache"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/services"
	"github.com/Abhi1264/unicore/api/internal/ws"
)

type Deps struct {
	Pool          *db.Pool
	Tokens        *auth.TokenService
	Redis         *cache.Client
	Log           *slog.Logger
	StoragePath   string
	Hub           *ws.Hub
	Auth          *services.AuthService
	Admin         *services.AdminService
	Results       *services.ResultsService
	Academic      *services.AcademicService
	Fees          *services.FeesService
	Attendance    *services.AttendanceService
	Announcements *services.AnnouncementsService
	Documents     *services.DocumentsService
}
