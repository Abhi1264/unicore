package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/Abhi1264/unicore/api/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func testConfig(env string) *config.Config {
	return &config.Config{
		AppEnv:       env,
		BaseDomain:   "unicore.app",
		PlatformHost: "app.unicore.app",
	}
}

func TestIsPlatformHost(t *testing.T) {
	prod := testConfig("production")
	if !isPlatformHost(prod, "app.unicore.app") {
		t.Fatal("the configured platform host must be recognised")
	}
	for _, host := range []string{"bitmesra.unicore.app", "unicore.app", "localhost", "evil.com"} {
		if isPlatformHost(prod, host) {
			t.Fatalf("%q must not count as the platform host in production", host)
		}
	}

	dev := testConfig("development")
	for _, host := range []string{"localhost", "127.0.0.1", "unicore.app"} {
		if !isPlatformHost(dev, host) {
			t.Fatalf("%q should reach the console in development", host)
		}
	}
	if isPlatformHost(dev, "bitmesra.unicore.app") {
		t.Fatal("a tenant subdomain is never the platform host")
	}
}

func TestValidSlug(t *testing.T) {
	valid := []string{"bitmesra", "iit-delhi", "a12", "x-9"}
	for _, s := range valid {
		if !validSlug(s) {
			t.Fatalf("%q should be a valid slug", s)
		}
	}
	// Rejecting these before the lookup keeps a hostile Host header from
	// reaching the database and keeps the slug rules aligned with the
	// tenants_slug_format constraint.
	invalid := []string{
		"", "a", "ab", "-lead", "trail-", "Upper", "under_score", "dot.ted",
		"semi;colon", "space bar", "sql'inject", "../etc", "a" + string(make([]byte, 70)),
	}
	for _, s := range invalid {
		if validSlug(s) {
			t.Fatalf("%q should be rejected", s)
		}
	}
}

func TestValidRequestID(t *testing.T) {
	if !validRequestID("abc-123_XYZ") {
		t.Fatal("a plain token should be accepted")
	}
	for _, id := range []string{"", "has space", "new\nline", string(make([]byte, 65))} {
		if validRequestID(id) {
			t.Fatalf("%q should be rejected", id)
		}
	}
}

// authApp wires Authenticate behind a stub that fakes whatever ResolveTenant
// would have produced, so the binding logic can be tested on its own.
func authApp(tokens *auth.TokenService, tenant *TenantInfo, platform bool) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		if platform {
			c.Locals(KeyPlatformHost, true)
		}
		if tenant != nil {
			c.Locals(KeyTenant, *tenant)
		}
		return c.Next()
	})
	app.Use(Authenticate(tokens))
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	return app
}

func do(t *testing.T, app *fiber.App, token string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/x", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func tokenService() *auth.TokenService {
	return auth.NewTokenService(
		"middleware-test-access-secret-long-enough",
		"middleware-test-refresh-secret-long-enough",
		time.Minute, time.Hour,
	)
}

// The core multi-tenancy assertion: a token minted for institute A must not be
// accepted on institute B's host, even though it is perfectly well signed.
func TestAuthenticateRejectsTokenFromAnotherTenant(t *testing.T) {
	tokens := tokenService()
	tenantA, tenantB := uuid.New(), uuid.New()
	tok, _, err := tokens.IssueAccess(uuid.New(), tenantA, auth.RoleStudent, "s@a.edu")
	if err != nil {
		t.Fatal(err)
	}

	app := authApp(tokens, &TenantInfo{ID: tenantB, Slug: "b", Status: "active"}, false)
	if got := do(t, app, tok); got != fiber.StatusForbidden {
		t.Fatalf("cross-tenant token got %d, want 403", got)
	}

	same := authApp(tokens, &TenantInfo{ID: tenantA, Slug: "a", Status: "active"}, false)
	if got := do(t, same, tok); got != fiber.StatusOK {
		t.Fatalf("same-tenant token got %d, want 200", got)
	}
}

// A superadmin token is a platform credential; replaying it against a tenant
// subdomain must not grant access to that tenant's data.
func TestAuthenticateConfinesSuperadminToPlatformHost(t *testing.T) {
	tokens := tokenService()
	tok, _, err := tokens.IssueAccess(uuid.New(), uuid.New(), auth.RoleSuperadmin, "root@unicore.app")
	if err != nil {
		t.Fatal(err)
	}

	onTenant := authApp(tokens, &TenantInfo{ID: uuid.New(), Slug: "b", Status: "active"}, false)
	if got := do(t, onTenant, tok); got != fiber.StatusForbidden {
		t.Fatalf("superadmin on a tenant host got %d, want 403", got)
	}

	onPlatform := authApp(tokens, nil, true)
	if got := do(t, onPlatform, tok); got != fiber.StatusOK {
		t.Fatalf("superadmin on the platform host got %d, want 200", got)
	}
}

// A tenant-scoped token on the bare platform host has no tenant to bind to.
// Letting it through would leave handlers with an unscoped request.
func TestAuthenticateRequiresResolvedTenantForTenantRoles(t *testing.T) {
	tokens := tokenService()
	tok, _, err := tokens.IssueAccess(uuid.New(), uuid.New(), auth.RoleInstituteAdmin, "a@b.edu")
	if err != nil {
		t.Fatal(err)
	}
	app := authApp(tokens, nil, true)
	if got := do(t, app, tok); got != fiber.StatusBadRequest {
		t.Fatalf("got %d, want 400", got)
	}
}

func TestAuthenticateRejectsMissingAndMalformedTokens(t *testing.T) {
	tokens := tokenService()
	tenant := &TenantInfo{ID: uuid.New(), Slug: "a", Status: "active"}
	for _, tok := range []string{"", "not-a-jwt", "a.b.c"} {
		app := authApp(tokens, tenant, false)
		if got := do(t, app, tok); got != fiber.StatusUnauthorized {
			t.Fatalf("token %q got %d, want 401", tok, got)
		}
	}
}

func TestRequirePlatformHost(t *testing.T) {
	build := func(platform bool) *fiber.App {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			if platform {
				c.Locals(KeyPlatformHost, true)
			}
			return c.Next()
		})
		app.Use(RequirePlatformHost())
		app.Get("/x", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
		return app
	}
	if got := do(t, build(false), ""); got != fiber.StatusNotFound {
		t.Fatalf("tenant host got %d, want 404", got)
	}
	if got := do(t, build(true), ""); got != fiber.StatusOK {
		t.Fatalf("platform host got %d, want 200", got)
	}
}

func TestRequireRoles(t *testing.T) {
	build := func(role auth.Role, allowed ...auth.Role) *fiber.App {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			if role != "" {
				c.Locals(KeyClaims, &auth.Claims{UserID: uuid.New(), Role: role})
			}
			return c.Next()
		})
		app.Use(RequireRoles(allowed...))
		app.Get("/x", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
		return app
	}

	if got := do(t, build(auth.RoleInstituteAdmin, auth.RoleInstituteAdmin), ""); got != fiber.StatusOK {
		t.Fatalf("admin on an admin route got %d, want 200", got)
	}
	// Privilege escalation check: lower roles must be turned away server-side,
	// not merely hidden in the UI.
	for _, role := range []auth.Role{auth.RoleStudent, auth.RoleFaculty} {
		if got := do(t, build(role, auth.RoleInstituteAdmin), ""); got != fiber.StatusForbidden {
			t.Fatalf("%s on an admin route got %d, want 403", role, got)
		}
	}
	if got := do(t, build("", auth.RoleInstituteAdmin), ""); got != fiber.StatusUnauthorized {
		t.Fatalf("unauthenticated got %d, want 401", got)
	}
}
