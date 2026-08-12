package services_test

import (
	"testing"

	"github.com/Abhi1264/unicore/api/internal/services"
)

func ptr(f float64) *float64 { return &f }

func TestComputeCumulativeCGPA(t *testing.T) {
	val, display := services.ComputeCumulative("cgpa", []services.GradeInput{
		{Grade: "A", GradePoints: ptr(9), Credits: 4},
		{Grade: "B", GradePoints: ptr(7), Credits: 2},
	})
	// (9*4 + 7*2) / 6 = 50/6 = 8.333 -> 8.33
	if val != 8.33 {
		t.Fatalf("expected 8.33, got %v", val)
	}
	if display != "8.33" {
		t.Fatalf("display %q", display)
	}
}

func TestComputeCumulativePercentage(t *testing.T) {
	val, display := services.ComputeCumulative("percentage", []services.GradeInput{
		{Marks: ptr(80), Credits: 3},
		{Marks: ptr(90), Credits: 3},
	})
	if val != 85 {
		t.Fatalf("expected 85, got %v", val)
	}
	if display != "85.00%" {
		t.Fatalf("display %q", display)
	}
}

func TestComputeCumulativeLetter(t *testing.T) {
	_, display := services.ComputeCumulative("letter", []services.GradeInput{
		{GradePoints: ptr(9.5), Credits: 3},
	})
	if display != "O" {
		t.Fatalf("expected O, got %q", display)
	}
}
