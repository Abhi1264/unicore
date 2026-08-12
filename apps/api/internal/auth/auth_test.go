package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Abhi1264/unicore/api/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	testAccessSecret  = "test-access-secret-that-is-long-enough"
	testRefreshSecret = "test-refresh-secret-that-is-long-enough"
	goodPassword      = "correct horse battery"
)

func newService(t *testing.T) *auth.TokenService {
	t.Helper()
	return auth.NewTokenService(testAccessSecret, testRefreshSecret, time.Minute, time.Hour)
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := auth.HashPassword(goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !auth.CheckPassword(hash, goodPassword) {
		t.Fatal("expected match")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected mismatch")
	}
}

func TestPasswordPolicy(t *testing.T) {
	if _, err := auth.HashPassword("short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
	// bcrypt truncates at 72 bytes; accepting a longer password would silently
	// ignore everything past the cut.
	if _, err := auth.HashPassword(strings.Repeat("a", 100)); err == nil {
		t.Fatal("expected over-long password to be rejected rather than truncated")
	}
}

func TestCheckPasswordConstantTimeRejectsEmptyHash(t *testing.T) {
	if auth.CheckPasswordConstantTime("", goodPassword) {
		t.Fatal("an absent hash must never authenticate")
	}
}

func TestAccessTokenClaims(t *testing.T) {
	ts := newService(t)
	uid, tid := uuid.New(), uuid.New()
	tok, _, err := ts.IssueAccess(uid, tid, auth.RoleStudent, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ts.ParseAccess(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != uid || claims.TenantID != tid || claims.Role != auth.RoleStudent {
		t.Fatalf("claims mismatch: %+v", claims)
	}
}

// A token signed with "alg": "none" must be rejected outright. Accepting it
// would let anyone mint a superadmin identity with no key at all.
func TestParseAccessRejectsAlgNone(t *testing.T) {
	ts := newService(t)
	claims := auth.Claims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Role:     auth.RoleSuperadmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "unicore",
			Subject:   uuid.NewString(),
			Audience:  []string{"unicore:access"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.ParseAccess(tok); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}

func TestParseAccessRejectsWrongSecret(t *testing.T) {
	other := auth.NewTokenService("a-completely-different-access-secret", testRefreshSecret, time.Minute, time.Hour)
	tok, _, err := other.IssueAccess(uuid.New(), uuid.New(), auth.RoleStudent, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newService(t).ParseAccess(tok); err == nil {
		t.Fatal("token signed with a foreign secret was accepted")
	}
}

// The two token types are separated by audience as well as by key, so neither
// can stand in for the other even if the secrets were ever misconfigured.
func TestRefreshTokenCannotBeUsedAsAccessToken(t *testing.T) {
	ts := newService(t)
	refresh, _, _, err := ts.IssueRefresh(uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.ParseAccess(refresh); err == nil {
		t.Fatal("refresh token was accepted as an access token")
	}
}

func TestAccessTokenCannotBeUsedAsRefreshToken(t *testing.T) {
	ts := newService(t)
	access, _, err := ts.IssueAccess(uuid.New(), uuid.New(), auth.RoleStudent, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.ParseRefresh(access); err == nil {
		t.Fatal("access token was accepted as a refresh token")
	}
}

func TestParseAccessRejectsExpiredToken(t *testing.T) {
	ts := auth.NewTokenService(testAccessSecret, testRefreshSecret, -time.Minute, time.Hour)
	tok, _, err := ts.IssueAccess(uuid.New(), uuid.New(), auth.RoleStudent, "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.ParseAccess(tok); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestParseAccessRejectsUnknownRole(t *testing.T) {
	ts := newService(t)
	tok, _, err := ts.IssueAccess(uuid.New(), uuid.New(), auth.Role("root"), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ts.ParseAccess(tok); err == nil {
		t.Fatal("token with an unrecognised role was accepted")
	}
}

func TestSecretEqual(t *testing.T) {
	if !auth.SecretEqual("abc", "abc") {
		t.Fatal("equal secrets should compare equal")
	}
	if auth.SecretEqual("abc", "abd") || auth.SecretEqual("abc", "ab") {
		t.Fatal("unequal secrets should not compare equal")
	}
}

func TestVerifyHMACSignature(t *testing.T) {
	body := []byte(`{"payment_id":"x"}`)
	// Signature produced by the same construction the verifier uses.
	const secret = "webhook-secret"
	good := hmacHex(t, secret, body)

	if !auth.VerifyHMACSignature(secret, body, good) {
		t.Fatal("valid signature rejected")
	}
	if auth.VerifyHMACSignature(secret, []byte(`{"payment_id":"y"}`), good) {
		t.Fatal("signature accepted for a different body")
	}
	if auth.VerifyHMACSignature("other-secret", body, good) {
		t.Fatal("signature accepted under the wrong secret")
	}
	if auth.VerifyHMACSignature("", body, good) || auth.VerifyHMACSignature(secret, body, "") {
		t.Fatal("empty secret or signature must never verify")
	}
}
