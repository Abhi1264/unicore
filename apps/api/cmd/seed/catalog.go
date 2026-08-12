package main

type deptSpec struct {
	Code, Name string
}

type courseSpec struct {
	Code, Name, Dept string
	Credits          float64
}

type feeSpec struct {
	Name    string
	Amount  float64
	LateFee float64
	DueDays int // relative to now; negative = overdue
}

type announcementSpec struct {
	Title, Body, Scope string
}

var departments = []deptSpec{
	{"CSE", "Computer Science"},
	{"ECE", "Electronics"},
	{"ME", "Mechanical Engineering"},
	{"MATH", "Mathematics"},
}

var courseCatalog = []courseSpec{
	{"CS101", "Data Structures", "CSE", 4},
	{"CS201", "Algorithms", "CSE", 4},
	{"CS220", "Database Systems", "CSE", 3},
	{"CS301", "Operating Systems", "CSE", 4},
	{"CS310", "Computer Networks", "CSE", 3},
	{"EC101", "Digital Logic", "ECE", 3},
	{"EC210", "Signals and Systems", "ECE", 4},
	{"ME101", "Engineering Mechanics", "ME", 3},
	{"ME220", "Thermodynamics", "ME", 4},
	{"MA101", "Calculus I", "MATH", 4},
	{"MA201", "Linear Algebra", "MATH", 3},
	{"MA301", "Probability and Statistics", "MATH", 3},
}

var feeCatalog = []feeSpec{
	{"Tuition", 50000, 1000, 21},
	{"Hostel", 18000, 500, 14},
	{"Exam Fee", 2500, 200, 45},
	{"Library", 800, 0, -7}, // overdue on purpose
}

var announcementCatalog = []announcementSpec{
	{"Welcome to the new semester", "Classes begin Monday. Check the timetable and complete fee payment before the deadline.", "all"},
	{"Registration window open", "Course registration is open for 2026S1. Seat caps are enforced — register early for popular electives.", "all"},
	{"Fee payment reminder", "Tuition and hostel dues are visible under Fees. Use the Pay action; do not share payment links.", "all"},
	{"Mid-semester exam schedule", "Midsems start in three weeks. Draft seating and room allotment will be posted here.", "all"},
	{"Library overdue notices", "Return overdue books to avoid blocks on result publishing.", "all"},
	{"Hackathon this Saturday", "CSE department is hosting a 24-hour campus hackathon. Register with your faculty mentor.", "program"},
	{"Lab safety briefing", "ME workshop users must complete the safety module before using heavy equipment.", "program"},
	{"Result day logistics", "Published results appear under Results. Contact your course faculty for grade queries within 7 days.", "all"},
}

var firstNames = []string{
	"Aarav", "Vivaan", "Aditya", "Vihaan", "Arjun", "Sai", "Reyansh", "Ayaan",
	"Krishna", "Ishaan", "Shaurya", "Atharv", "Advait", "Dhruv", "Kabir", "Ritvik",
	"Ananya", "Aadhya", "Aarohi", "Diya", "Myra", "Anika", "Sara", "Ira",
	"Kiara", "Navya", "Pari", "Saanvi", "Riya", "Meera", "Isha", "Tara",
}

var lastNames = []string{
	"Sharma", "Verma", "Patel", "Reddy", "Nair", "Iyer", "Khan", "Singh",
	"Das", "Ghosh", "Banerjee", "Mehta", "Joshi", "Gupta", "Malhotra", "Chopra",
}
