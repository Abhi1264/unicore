package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleStudent        Role = "student"
	RoleFaculty        Role = "faculty"
	RoleInstituteAdmin Role = "institute_admin"
	RoleSuperadmin     Role = "superadmin"
)

func (r Role) Valid() bool {
	switch r {
	case RoleStudent, RoleFaculty, RoleInstituteAdmin, RoleSuperadmin:
		return true
	}
	return false
}

const (
	// bcrypt silently truncates input at 72 bytes, so anything longer would make
	// the tail of a long passphrase meaningless. Reject instead of truncating.
	MaxPasswordBytes = 72
	MinPasswordRunes = 12
	// Cost 12 is roughly 250ms on current server hardware — slow enough to make
	// offline cracking expensive, fast enough for an interactive login.
	bcryptCost = 12
)

var (
	ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", MinPasswordRunes)
	ErrPasswordTooLong  = fmt.Errorf("password must be at most %d bytes", MaxPasswordBytes)
)

// dummyHash is compared against when a login names an account that does not
// exist, so a missing user costs the same wall-clock time as a wrong password
// and the endpoint cannot be used to enumerate valid emails.
var dummyHash = mustHash("unicore-timing-equaliser")

func mustHash(pw string) []byte {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		panic(err)
	}
	return h
}

func ValidatePassword(pw string) error {
	if len(pw) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	if utf8.RuneCountInString(pw) < MinPasswordRunes {
		return ErrPasswordTooShort
	}
	return nil
}

func HashPassword(pw string) (string, error) {
	if err := ValidatePassword(pw); err != nil {
		return "", err
	}
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// CheckPasswordConstantTime burns the same amount of time as a real comparison
// when hash is empty. Callers use it on the "user not found" path.
func CheckPasswordConstantTime(hash, pw string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false
	}
	return CheckPassword(hash, pw)
}

// SecretEqual compares two secrets without leaking their common prefix length
// through timing. Use it for webhook signatures, invite codes and API keys —
// anywhere `==` would otherwise short-circuit on the first differing byte.
func SecretEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// VerifyHMACSignature checks a hex-encoded HMAC-SHA256 of body against secret.
func VerifyHMACSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return SecretEqual(fmt.Sprintf("%x", mac.Sum(nil)), signature)
}

type Claims struct {
	UserID   uuid.UUID `json:"uid"`
	TenantID uuid.UUID `json:"tenant_id"`
	Role     Role      `json:"role"`
	Email    string    `json:"email"`
	jwt.RegisteredClaims
}

type TokenService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

const (
	tokenIssuer      = "unicore"
	audienceAccess   = "unicore:access"
	audienceRefresh  = "unicore:refresh"
)

func (t *TokenService) IssueAccess(userID, tenantID uuid.UUID, role Role, email string) (string, time.Time, error) {
	exp := time.Now().Add(t.accessTTL)
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   userID.String(),
			ID:        uuid.NewString(),
			Audience:  []string{audienceAccess},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-30 * time.Second)),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.accessSecret)
	return s, exp, err
}

// IssueRefresh mints a refresh token. Tenant identity travels in a private claim
// rather than `aud`, because `aud` is reserved for separating the two token
// types — an access token must never validate as a refresh token or vice versa.
func (t *TokenService) IssueRefresh(userID, tenantID uuid.UUID) (string, string, time.Time, error) {
	exp := time.Now().Add(t.refreshTTL)
	jti := uuid.NewString()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   userID.String(),
			ID:        jti,
			Audience:  []string{audienceRefresh},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.refreshSecret)
	return s, jti, exp, err
}

var errBadToken = errors.New("invalid token")

func (t *TokenService) parse(token string, secret []byte, audience string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{},
		func(*jwt.Token) (any, error) { return secret, nil },
		// Pinning the algorithm at the parser level is what makes `alg: none` and
		// the HS256/RS256 confusion attack impossible; the keyfunc is never even
		// consulted for a token signed with anything else.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, errBadToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errBadToken
	}
	if claims.UserID == uuid.Nil {
		return nil, errBadToken
	}
	return claims, nil
}

func (t *TokenService) ParseAccess(token string) (*Claims, error) {
	claims, err := t.parse(token, t.accessSecret, audienceAccess)
	if err != nil {
		return nil, err
	}
	if !claims.Role.Valid() || claims.TenantID == uuid.Nil {
		return nil, errBadToken
	}
	return claims, nil
}

func (t *TokenService) ParseRefresh(token string) (*Claims, error) {
	claims, err := t.parse(token, t.refreshSecret, audienceRefresh)
	if err != nil {
		return nil, err
	}
	if claims.ID == "" {
		return nil, errBadToken
	}
	return claims, nil
}

func (t *TokenService) RefreshTTL() time.Duration { return t.refreshTTL }
