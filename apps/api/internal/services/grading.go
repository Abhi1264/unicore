package services

import (
	"fmt"
	"math"
	"strings"
)

// GradeInput is one graded course contribution toward a cumulative.
type GradeInput struct {
	Grade       string
	GradePoints *float64
	Marks       *float64
	Credits     float64
}

// ComputeCumulative computes a cumulative value and display string for the
// tenant grading system: cgpa (credit-weighted grade points), percentage
// (average marks), or letter (GPA-like from points, shown as a letter).
func ComputeCumulative(gradingSystem string, rows []GradeInput) (value float64, display string) {
	switch strings.ToLower(strings.TrimSpace(gradingSystem)) {
	case "percentage":
		return computePercentage(rows)
	case "letter":
		gpa, _ := computeWeightedGPA(rows)
		return gpa, letterFromGPA(gpa)
	default: // cgpa
		gpa, ok := computeWeightedGPA(rows)
		if !ok {
			return 0, "0.00"
		}
		return gpa, fmt.Sprintf("%.2f", gpa)
	}
}

func computeWeightedGPA(rows []GradeInput) (float64, bool) {
	var weighted, credits float64
	for _, r := range rows {
		if r.GradePoints == nil || r.Credits <= 0 {
			continue
		}
		weighted += *r.GradePoints * r.Credits
		credits += r.Credits
	}
	if credits == 0 {
		return 0, false
	}
	return round2(weighted / credits), true
}

func computePercentage(rows []GradeInput) (float64, string) {
	var sum float64
	var n int
	for _, r := range rows {
		if r.Marks == nil {
			continue
		}
		sum += *r.Marks
		n++
	}
	if n == 0 {
		return 0, "0.00%"
	}
	avg := round2(sum / float64(n))
	return avg, fmt.Sprintf("%.2f%%", avg)
}

func letterFromGPA(gpa float64) string {
	switch {
	case gpa >= 9.0:
		return "O"
	case gpa >= 8.0:
		return "A+"
	case gpa >= 7.0:
		return "A"
	case gpa >= 6.0:
		return "B+"
	case gpa >= 5.0:
		return "B"
	case gpa >= 4.0:
		return "C"
	default:
		return "F"
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
