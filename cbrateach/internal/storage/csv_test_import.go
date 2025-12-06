package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cbrateach/internal/models"
)

// ImportTestFromCSV imports a test from CSV file
// CSV format: Vorname,Nachname,Q1,Q2,Q3,...
// First row is headers
func (s *Storage) ImportTestFromCSV(csvPath, courseID, courseName, testName, testTopic string, weight float64) error {
	// Read CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file must have at least header row and one data row")
	}

	// Parse header row
	headers := records[0]
	if len(headers) < 3 {
		return fmt.Errorf("CSV must have at least: Vorname, Nachname, and one question column")
	}

	// Extract question columns (everything after Nachname)
	var questions []models.Question
	questionHeaders := headers[2:] // Skip Vorname, Nachname

	for i, qHeader := range questionHeaders {
		// Try to parse max points from header if format is like "Q1 (10)"
		maxPoints := 1.0 // Default
		title := strings.TrimSpace(qHeader)

		// Check for points in parentheses
		if strings.Contains(title, "(") && strings.Contains(title, ")") {
			start := strings.Index(title, "(")
			end := strings.Index(title, ")")
			if start < end {
				pointsStr := strings.TrimSpace(title[start+1 : end])
				if points, err := strconv.ParseFloat(pointsStr, 64); err == nil {
					maxPoints = points
					title = strings.TrimSpace(title[:start])
				}
			}
		}

		questions = append(questions, models.Question{
			ID:        fmt.Sprintf("q%d", i+1),
			Title:     title,
			MaxPoints: maxPoints,
		})
	}

	// Parse student scores
	var studentScores []models.StudentScore

	for i := 1; i < len(records); i++ {
		record := records[i]

		if len(record) < len(headers) {
			continue // Skip incomplete rows
		}

		vorname := strings.TrimSpace(record[0])
		nachname := strings.TrimSpace(record[1])

		if vorname == "" && nachname == "" {
			continue // Skip empty rows
		}

		fullName := fmt.Sprintf("%s %s", vorname, nachname)

		// Parse question scores
		questionScores := make(map[string]float64)
		for j, q := range questions {
			scoreStr := strings.TrimSpace(record[2+j])
			score := 0.0

			if scoreStr != "" {
				if parsedScore, err := strconv.ParseFloat(scoreStr, 64); err == nil {
					score = parsedScore
				}
			}

			questionScores[q.ID] = score
		}

		studentScore := models.StudentScore{
			StudentName:    fullName,
			QuestionScores: questionScores,
		}

		studentScores = append(studentScores, studentScore)
	}

	// Create test
	if weight <= 0 {
		weight = 1.0 // Default weight
	}

	test := models.Test{
		ID:            GenerateID(),
		CourseID:      courseID,
		CourseName:    courseName,
		Title:         testName,
		Topic:         testTopic,
		Date:          time.Now(),
		Questions:     questions,
		StudentScores: studentScores,
		GiftedPoints:  0,
		Weight:        weight,
		Status:        "review",
	}

	// Calculate grades
	s.RecalculateTestGrades(&test)

	// Save test
	if err := s.AddTest(test); err != nil {
		return fmt.Errorf("failed to save test: %w", err)
	}

	fmt.Printf("Successfully imported test '%s' for course '%s'\n", testName, courseName)
	fmt.Printf("  %d questions, %d students\n", len(questions), len(studentScores))

	return nil
}
