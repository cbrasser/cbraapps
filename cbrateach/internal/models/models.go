package models

import "time"

type Mark struct {
	Date   time.Time `json:"date"`
	Reason string    `json:"reason"`
}

type Student struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Note          string `json:"note,omitempty"`
	PositiveMarks []Mark `json:"positive_marks,omitempty"`
	NegativeMarks []Mark `json:"negative_marks,omitempty"`
}

type Course struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Subject      string    `json:"subject"`
	Weekday      string    `json:"weekday"`
	Time         string    `json:"time"`
	Room         string    `json:"room"`
	CurrentTopic string    `json:"current_topic"`
	Students     []Student `json:"students"`
	NoteFile     string    `json:"note_file"` // Path to markdown note file
}

type ReviewStudent struct {
	Name     string `json:"name"`
	Positive bool   `json:"positive"` // true for positive, false for negative
	Reason   string `json:"reason"`
}

type Review struct {
	ID             string          `json:"id"`
	CourseID       string          `json:"course_id"`
	CourseName     string          `json:"course_name"`
	Date           time.Time       `json:"date"`
	Title          string          `json:"title"`
	Topic          string          `json:"topic"`
	ReviewText     string          `json:"review_text,omitempty"`
	StudentsStandOut []ReviewStudent `json:"students_stand_out,omitempty"`
}

// Test-related models

type Question struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`      // e.g., "Q1", "Question 1"
	MaxPoints float64 `json:"max_points"` // Maximum points for this question
}

type StudentScore struct {
	StudentName    string             `json:"student_name"` // Full name
	QuestionScores map[string]float64 `json:"question_scores"` // questionID -> points scored
	QuestionComments map[string]string `json:"question_comments"` // questionID -> comment
	TotalPoints    float64            `json:"total_points"`    // Calculated
	Grade          float64            `json:"grade"`           // Calculated (1.0 to 6.0, rounded to 0.25)
}

type Test struct {
	ID            string         `json:"id"`
	CourseID      string         `json:"course_id"`
	CourseName    string         `json:"course_name"`
	Title         string         `json:"title"`
	Topic         string         `json:"topic"`
	Date          time.Time      `json:"date"`
	Questions     []Question     `json:"questions"`
	StudentScores []StudentScore `json:"student_scores"`
	GiftedPoints  float64        `json:"gifted_points"` // Points subtracted from max for grade calculation
	Weight        float64        `json:"weight"`        // Weight for final grade calculation (default 1.0)
	Status        string         `json:"status"`        // "review" or "confirmed"
}

// CalculateTotalPoints calculates total points for a student
func (ss *StudentScore) CalculateTotalPoints() {
	total := 0.0
	for _, points := range ss.QuestionScores {
		total += points
	}
	ss.TotalPoints = total
}

// CalculateGrade calculates grade based on points
// Formula: (points / (max_points - gifted_points)) * 5 + 1
// Rounded to quarters (0.25)
func (t *Test) CalculateGrade(studentScore *StudentScore) float64 {
	maxPoints := 0.0
	for _, q := range t.Questions {
		maxPoints += q.MaxPoints
	}

	adjustedMax := maxPoints - t.GiftedPoints
	if adjustedMax <= 0 {
		return 1.0 // Avoid division by zero
	}

	grade := (studentScore.TotalPoints / adjustedMax) * 5.0 + 1.0

	// Round to nearest quarter
	grade = roundToQuarter(grade)

	// Clamp between 1.0 and 6.0
	if grade < 1.0 {
		grade = 1.0
	}
	if grade > 6.0 {
		grade = 6.0
	}

	return grade
}

func roundToQuarter(val float64) float64 {
	return float64(int(val*4+0.5)) / 4.0
}
