package services

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTenantNotActive    = errors.New("tenant not active")
	ErrSeatFull           = errors.New("course seat capacity reached")
	ErrInvalidInput       = errors.New("invalid input")
	ErrRegistrationClosed = errors.New("registration window closed")
)

func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func TextOrEmpty(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func UUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func UUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func NumericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(f, 'f', -1, 64))
	return n
}

func NumericFromFloatPtr(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	return NumericFromFloat(*f)
}

func FloatFromNumeric(n pgtype.Numeric) (float64, bool) {
	if !n.Valid {
		return 0, false
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0, false
	}
	return f.Float64, true
}

func DateFromTime(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t.UTC().Truncate(24 * time.Hour), Valid: true}
}

// TimeOfDay builds a pgtype.Time from clock components (24h).
func TimeOfDay(hour, minute, second int) pgtype.Time {
	us := int64(hour)*3_600_000_000 + int64(minute)*60_000_000 + int64(second)*1_000_000
	return pgtype.Time{Microseconds: us, Valid: true}
}

// Postgres codes mapped to domain errors (prefer constraints over pre-flight SELECTs).
const (
	pgUniqueViolation     = "23505"
	pgCheckViolation      = "23514"
	pgForeignKeyViolation = "23503"
)

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func isUniqueViolation(err error) bool { return pgErrorCode(err) == pgUniqueViolation }
func isCheckViolation(err error) bool  { return pgErrorCode(err) == pgCheckViolation }
func isForeignKeyViolation(err error) bool {
	return pgErrorCode(err) == pgForeignKeyViolation
}

func fmtErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
