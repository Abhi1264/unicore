package pdf

import (
	"bytes"
	"testing"
	"time"
)

func TestGenerateMarksheetIncludesLiveRows(t *testing.T) {
	b, err := GenerateMarksheet(MarksheetInput{
		InstituteName: "Demo College",
		StudentName:   "Ada Lovelace",
		RollNumber:    "CSE001",
		Program:       "B.Tech CSE",
		BatchYear:     2024,
		GeneratedAt:   time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
		Rows: []MarksheetRow{
			{Code: "CS101", Name: "Programming", Semester: "2026S1", Grade: "A", Credits: "4.0", Marks: "88"},
		},
		Cumulative: "8.50",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatal("expected a PDF")
	}
	for _, want := range []string{"Ada Lovelace", "CSE001", "CS101", "8.50"} {
		if !bytes.Contains(b, []byte(want)) {
			t.Fatalf("pdf missing %q", want)
		}
	}
}

func TestGenerateBonafideIncludesStudent(t *testing.T) {
	b, err := GenerateBonafide(BonafideInput{
		InstituteName: "Demo College",
		StudentName:   "Ada Lovelace",
		RollNumber:    "CSE001",
		Program:       "B.Tech CSE",
		BatchYear:     2024,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte("bona fide student")) {
		t.Fatal("expected bonafide wording")
	}
	if !bytes.Contains(b, []byte("Ada Lovelace")) {
		t.Fatal("missing student name")
	}
}
