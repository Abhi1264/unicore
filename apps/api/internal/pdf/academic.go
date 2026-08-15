package pdf

import (
	"fmt"
	"strings"
	"time"
)

type MarksheetInput struct {
	InstituteName string
	StudentName   string
	RollNumber    string
	Program       string
	BatchYear     int32
	GeneratedAt   time.Time
	Rows          []MarksheetRow
	Cumulative    string
}

type MarksheetRow struct {
	Code     string
	Name     string
	Semester string
	Grade    string
	Credits  string
	Marks    string
}

type BonafideInput struct {
	InstituteName string
	StudentName   string
	RollNumber    string
	Program       string
	BatchYear     int32
	GeneratedAt   time.Time
}

func GenerateMarksheet(in MarksheetInput) ([]byte, error) {
	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}
	lines := []string{
		"UNICORE MARKSHEET",
		"",
		"Institute: " + nullSafe(in.InstituteName),
		"Student: " + nullSafe(in.StudentName),
		"Roll: " + nullSafe(in.RollNumber),
		"Program: " + nullSafe(in.Program),
		fmt.Sprintf("Batch: %d", in.BatchYear),
		"Issued: " + when.UTC().Format("02 Jan 2006"),
		"",
		"Code  Course                         Sem  Grade  Cr  Marks",
		strings.Repeat("-", 58),
	}
	if len(in.Rows) == 0 {
		lines = append(lines, "No published results.")
	}
	for _, r := range in.Rows {
		lines = append(lines, fmt.Sprintf("%-5s %-28s %-4s %-6s %-3s %s",
			clip(r.Code, 5), clip(r.Name, 28), clip(r.Semester, 4), clip(r.Grade, 6), clip(r.Credits, 3), nullSafe(r.Marks)))
	}
	if in.Cumulative != "" {
		lines = append(lines, "", "Cumulative: "+in.Cumulative)
	}
	lines = append(lines, "", "This is a computer-generated statement of published results.")
	return WriteSimplePDF(lines)
}

func GenerateBonafide(in BonafideInput) ([]byte, error) {
	when := in.GeneratedAt
	if when.IsZero() {
		when = time.Now().UTC()
	}
	lines := []string{
		"BONAFIDE CERTIFICATE",
		"",
		"This is to certify that " + nullSafe(in.StudentName) + ",",
		"Roll No. " + nullSafe(in.RollNumber) + ",",
		"is a bona fide student of " + nullSafe(in.InstituteName) + ",",
		"enrolled in " + nullSafe(in.Program) + fmt.Sprintf(" (batch %d).", in.BatchYear),
		"",
		"Issued on " + when.UTC().Format("02 January 2006") + ".",
		"",
		"Registrar / Academic Office",
		nullSafe(in.InstituteName),
	}
	return WriteSimplePDF(lines)
}

func clip(s string, n int) string {
	s = nullSafe(s)
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "."
}
