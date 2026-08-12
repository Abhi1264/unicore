package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/cache"
	"github.com/Abhi1264/unicore/api/internal/db"
	"github.com/Abhi1264/unicore/api/internal/db/sqlcdb"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthService struct {
	pool   *db.Pool
	cache  *cache.Client
	tokens *auth.TokenService
}

func NewAuthService(pool *db.Pool, cacheClient *cache.Client, tokens *auth.TokenService) *AuthService {
	return &AuthService{pool: pool, cache: cacheClient, tokens: tokens}
}

type RegisterTenantResult struct {
	Tenant sqlcdb.Tenant `json:"tenant"`
	Admin  sqlcdb.User   `json:"admin"`
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type LoginResult struct {
	User   sqlcdb.User `json:"user"`
	Tokens TokenPair   `json:"tokens"`
}

const (
	maxNameLength  = 200
	maxEmailLength = 254
)

// reservedSlugs mirrors tenants_slug_not_reserved for clear validation errors.
var reservedSlugs = map[string]struct{}{
	"www": {}, "api": {}, "app": {}, "admin": {}, "auth": {}, "login": {},
	"static": {}, "assets": {}, "cdn": {}, "mail": {}, "smtp": {}, "ftp": {},
	"ns1": {}, "ns2": {}, "status": {}, "docs": {}, "help": {}, "support": {},
	"blog": {}, "dashboard": {}, "internal": {}, "metrics": {}, "grafana": {},
	"prometheus": {}, "test": {}, "staging": {}, "dev": {}, "demo": {}, "unicore": {},
}

func ValidateSlug(slug string) error {
	if len(slug) < 3 || len(slug) > 63 {
		return fmt.Errorf("%w: subdomain must be 3-63 characters", ErrInvalidInput)
	}
	for i, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' && i > 0 && i < len(slug)-1:
		default:
			return fmt.Errorf("%w: subdomain may only contain lowercase letters, digits and inner hyphens", ErrInvalidInput)
		}
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return fmt.Errorf("%w: that subdomain is reserved", ErrInvalidInput)
	}
	return nil
}

func validateEmail(email string) error {
	if len(email) == 0 || len(email) > maxEmailLength {
		return fmt.Errorf("%w: email is required", ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("%w: email is not a valid address", ErrInvalidInput)
	}
	return nil
}

func validateName(field, v string) error {
	if v == "" || len(v) > maxNameLength {
		return fmt.Errorf("%w: %s must be 1-%d characters", ErrInvalidInput, field, maxNameLength)
	}
	return nil
}

func (s *AuthService) RegisterTenant(ctx context.Context, slug, name, adminEmail, adminPassword, adminName string) (RegisterTenantResult, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	name = strings.TrimSpace(name)
	adminEmail = strings.TrimSpace(strings.ToLower(adminEmail))
	adminName = strings.TrimSpace(adminName)

	if err := ValidateSlug(slug); err != nil {
		return RegisterTenantResult{}, err
	}
	if err := validateName("institute_name", name); err != nil {
		return RegisterTenantResult{}, err
	}
	if err := validateName("admin_full_name", adminName); err != nil {
		return RegisterTenantResult{}, err
	}
	if err := validateEmail(adminEmail); err != nil {
		return RegisterTenantResult{}, err
	}
	if err := auth.ValidatePassword(adminPassword); err != nil {
		return RegisterTenantResult{}, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return RegisterTenantResult{}, fmtErr("hash password", err)
	}

	// UNIQUE on slug is the race-safe check (not SELECT-then-INSERT).
	tenant, err := s.pool.Platform().CreateTenant(ctx, sqlcdb.CreateTenantParams{Slug: slug, Name: name})
	if err != nil {
		if isUniqueViolation(err) || isCheckViolation(err) {
			return RegisterTenantResult{}, fmt.Errorf("%w: subdomain is already taken", ErrConflict)
		}
		return RegisterTenantResult{}, fmtErr("create tenant", err)
	}

	var admin sqlcdb.User
	err = s.pool.WithTenant(ctx, tenant.ID, func(ctx context.Context, tq *sqlcdb.Queries) error {
		if _, err := tq.CreateTenantConfig(ctx, sqlcdb.CreateTenantConfigParams{
			TenantID:               tenant.ID,
			GradingSystem:          "cgpa",
			AcademicCalendarType:   "semester",
			Branding:               json.RawMessage(`{}`),
			GradingScale:           json.RawMessage(`{}`),
			AttendanceThresholdPct: NumericFromFloat(75),
		}); err != nil {
			return fmtErr("create tenant config", err)
		}
		admin, err = tq.CreateUser(ctx, sqlcdb.CreateUserParams{
			TenantID:     tenant.ID,
			Email:        adminEmail,
			PasswordHash: hash,
			Role:         string(auth.RoleInstituteAdmin),
			FullName:     adminName,
		})
		return fmtErr("create admin user", err)
	})
	if err != nil {
		return RegisterTenantResult{}, err
	}

	return RegisterTenantResult{Tenant: tenant, Admin: admin}, nil
}

func (s *AuthService) Login(ctx context.Context, tenantID uuid.UUID, email, password, ip, ua string) (LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" || len(email) > maxEmailLength || len(password) > auth.MaxPasswordBytes {
		return LoginResult{}, ErrInvalidCredentials
	}

	tenant, err := s.pool.Platform().GetTenantByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrNotFound
		}
		return LoginResult{}, fmtErr("get tenant", err)
	}
	if tenant.Status != "active" {
		return LoginResult{}, ErrTenantNotActive
	}

	var user sqlcdb.User
	var found bool
	err = s.pool.WithTenant(ctx, tenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		u, err := q.GetUserByEmail(ctx, sqlcdb.GetUserByEmailParams{TenantID: tenantID, Email: email})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		found = err == nil
		user = u

		// Always compare the hash, including for unknown emails (timing).
		if !auth.CheckPasswordConstantTime(user.PasswordHash, password) || !found || !user.IsActive {
			return ErrInvalidCredentials
		}
		return q.InsertLoginEvent(ctx, sqlcdb.InsertLoginEventParams{
			TenantID:  tenantID,
			UserID:    user.ID,
			Ip:        TextOrEmpty(ip),
			UserAgent: TextOrEmpty(truncate(ua, 512)),
		})
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}

	tokens, err := s.issueAndStore(ctx, user)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: user, Tokens: tokens}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}

	if s.revokedBefore(ctx, claims.UserID, claims.IssuedAt) {
		return TokenPair{}, ErrUnauthorized
	}

	// Stored jti is authoritative; rotation deletes it so replays miss.
	key := refreshKey(claims.ID)
	stored, err := s.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.Miss) {
			// Consumed/missing jti: revoke all sessions (replay or race).
			s.revokeAllForUser(ctx, claims.UserID)
			return TokenPair{}, ErrUnauthorized
		}
		return TokenPair{}, ErrUnauthorized
	}
	if !auth.SecretEqual(stored, refreshFingerprint(claims.UserID, claims.TenantID)) {
		s.revokeAllForUser(ctx, claims.UserID)
		return TokenPair{}, ErrUnauthorized
	}

	// Suspended tenants must stop minting access tokens within access-token TTL.
	tenant, err := s.pool.Platform().GetTenantByID(ctx, claims.TenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrUnauthorized
		}
		return TokenPair{}, fmtErr("get tenant", err)
	}
	if tenant.Status != "active" {
		return TokenPair{}, ErrTenantNotActive
	}

	var user sqlcdb.User
	err = s.pool.WithTenant(ctx, claims.TenantID, func(ctx context.Context, q *sqlcdb.Queries) error {
		var err error
		user, err = q.GetUserByID(ctx, sqlcdb.GetUserByIDParams{TenantID: claims.TenantID, ID: claims.UserID})
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrUnauthorized
		}
		return TokenPair{}, fmtErr("get user", err)
	}
	if !user.IsActive {
		return TokenPair{}, ErrForbidden
	}

	_ = s.cache.Del(ctx, key)
	return s.issueAndStore(ctx, user)
}

// Logout revokes the presented refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		// Do not confirm whether the token was valid.
		return nil
	}
	_ = s.cache.Del(ctx, refreshKey(claims.ID))
	return nil
}

func (s *AuthService) issueAndStore(ctx context.Context, user sqlcdb.User) (TokenPair, error) {
	access, exp, err := s.tokens.IssueAccess(user.ID, user.TenantID, auth.Role(user.Role), user.Email)
	if err != nil {
		return TokenPair{}, fmtErr("issue access", err)
	}
	refresh, jti, _, err := s.tokens.IssueRefresh(user.ID, user.TenantID)
	if err != nil {
		return TokenPair{}, fmtErr("issue refresh", err)
	}
	// Refresh requires a live jti; Redis outage must fail login.
	if err := s.cache.Set(ctx, refreshKey(jti), refreshFingerprint(user.ID, user.TenantID), s.tokens.RefreshTTL()); err != nil {
		return TokenPair{}, fmtErr("store refresh", err)
	}
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresAt: exp}, nil
}

// revokeAllForUser sets a revocation epoch consulted by revokedBefore on refresh.
func (s *AuthService) revokeAllForUser(ctx context.Context, userID uuid.UUID) {
	_ = s.cache.Set(ctx, revokedKey(userID), strconv.FormatInt(time.Now().UTC().UnixNano(), 10), s.tokens.RefreshTTL())
}

// revokedBefore reports whether issuedAt predates the user's revocation epoch.
// Cache errors (not Miss) fail closed so outages cannot resurrect killed sessions.
func (s *AuthService) revokedBefore(ctx context.Context, userID uuid.UUID, issuedAt *jwt.NumericDate) bool {
	raw, err := s.cache.Get(ctx, revokedKey(userID))
	if err != nil {
		return !errors.Is(err, cache.Miss)
	}
	epoch, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return true
	}
	if issuedAt == nil {
		return true
	}
	return issuedAt.UTC().UnixNano() < epoch
}

func refreshKey(jti string) string {
	return "refresh:" + jti
}

func revokedKey(userID uuid.UUID) string {
	return "refresh:revoked:" + userID.String()
}

func refreshFingerprint(userID, tenantID uuid.UUID) string {
	return userID.String() + "|" + tenantID.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
