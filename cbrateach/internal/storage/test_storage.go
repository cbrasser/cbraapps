package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cbrateach/internal/models"

	"github.com/xuri/excelize/v2"
)

// Test storage functions

func (s *Storage) TestsPath(courseID string) string {
	return filepath.Join(s.cfg.DataDir, fmt.Sprintf("tests_%s.json", courseID))
}

func (s *Storage) LoadTests(courseID string) ([]models.Test, error) {
	path := s.TestsPath(courseID)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Test{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tests []models.Test
	if err := json.Unmarshal(data, &tests); err != nil {
		return nil, err
	}

	return tests, nil
}

func (s *Storage) SaveTests(courseID string, tests []models.Test) error {
	data, err := json.MarshalIndent(tests, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.TestsPath(courseID), data, 0644)
}

func (s *Storage) AddTest(test models.Test) error {
	tests, err := s.LoadTests(test.CourseID)
	if err != nil {
		return err
	}

	tests = append(tests, test)
	return s.SaveTests(test.CourseID, tests)
}

func (s *Storage) UpdateTest(test models.Test) error {
	tests, err := s.LoadTests(test.CourseID)
	if err != nil {
		return err
	}

	for i := range tests {
		if tests[i].ID == test.ID {
			tests[i] = test
			return s.SaveTests(test.CourseID, tests)
		}
	}

	return fmt.Errorf("test not found: %s", test.ID)
}

func (s *Storage) GetTest(courseID, testID string) (*models.Test, error) {
	tests, err := s.LoadTests(courseID)
	if err != nil {
		return nil, err
	}

	for i := range tests {
		if tests[i].ID == testID {
			return &tests[i], nil
		}
	}

	return nil, fmt.Errorf("test not found: %s", testID)
}

// RecalculateTestGrades recalculates all grades for a test
func (s *Storage) RecalculateTestGrades(test *models.Test) {
	for i := range test.StudentScores {
		test.StudentScores[i].CalculateTotalPoints()
		test.StudentScores[i].Grade = test.CalculateGrade(&test.StudentScores[i])
	}
}

// ExportGrades exports average grades for all confirmed tests in a course
// Output format: Vorname,Nachname,Grade
func (s *Storage) ExportGrades(courseID, outputPath string) error {
	tests, err := s.LoadTests(courseID)
	if err != nil {
		return err
	}

	// Filter confirmed tests only
	var confirmedTests []models.Test
	for _, test := range tests {
		if test.Status == "confirmed" {
			confirmedTests = append(confirmedTests, test)
		}
	}

	if len(confirmedTests) == 0 {
		return fmt.Errorf("no confirmed tests found for this course")
	}

	// Calculate weighted average grade per student
	studentGrades := make(map[string]float64)  // student name -> total weighted grade
	studentWeights := make(map[string]float64) // student name -> total weight

	for _, test := range confirmedTests {
		weight := test.Weight
		if weight <= 0 {
			weight = 1.0
		}

		for _, score := range test.StudentScores {
			studentGrades[score.StudentName] += score.Grade * weight
			studentWeights[score.StudentName] += weight
		}
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write CSV
	_, _ = file.WriteString("Vorname,Nachname,Grade\n")

	for studentName, totalWeightedGrade := range studentGrades {
		totalWeight := studentWeights[studentName]
		avgGrade := totalWeightedGrade / totalWeight

		// Split name into first and last
		parts := strings.Fields(studentName)
		vorname := ""
		nachname := ""

		if len(parts) > 0 {
			vorname = parts[0]
		}
		if len(parts) > 1 {
			nachname = strings.Join(parts[1:], " ")
		}

		_, _ = file.WriteString(fmt.Sprintf("%s,%s,%.2f\n", vorname, nachname, avgGrade))
	}

	return nil
}

// ExportGradesXLSX exports average grades for all confirmed tests in XLSX format
func (s *Storage) ExportGradesXLSX(courseID, outputPath string) error {
	tests, err := s.LoadTests(courseID)
	if err != nil {
		return err
	}

	// Filter confirmed tests only
	var confirmedTests []models.Test
	for _, test := range tests {
		if test.Status == "confirmed" {
			confirmedTests = append(confirmedTests, test)
		}
	}

	if len(confirmedTests) == 0 {
		return fmt.Errorf("no confirmed tests found for this course")
	}

	// Calculate weighted average grade per student
	studentGrades := make(map[string]float64)
	studentWeights := make(map[string]float64)

	for _, test := range confirmedTests {
		weight := test.Weight
		if weight <= 0 {
			weight = 1.0
		}

		for _, score := range test.StudentScores {
			studentGrades[score.StudentName] += score.Grade * weight
			studentWeights[score.StudentName] += weight
		}
	}

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Final Grades"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}
	f.SetActiveSheet(index)

	// Set headers
	f.SetCellValue(sheetName, "A1", "Vorname")
	f.SetCellValue(sheetName, "B1", "Nachname")
	f.SetCellValue(sheetName, "C1", "Grade")

	// Write data
	row := 2
	for studentName, totalWeightedGrade := range studentGrades {
		totalWeight := studentWeights[studentName]
		avgGrade := totalWeightedGrade / totalWeight

		// Split name
		parts := strings.Fields(studentName)
		vorname := ""
		nachname := ""

		if len(parts) > 0 {
			vorname = parts[0]
		}
		if len(parts) > 1 {
			nachname = strings.Join(parts[1:], " ")
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), vorname)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), nachname)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), avgGrade)
		row++
	}

	// Delete default Sheet1
	f.DeleteSheet("Sheet1")

	// Save file
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("failed to save XLSX file: %w", err)
	}

	return nil
}

// DeleteTest removes a test from the course
func (s *Storage) DeleteTest(courseID, testID string) error {
	tests, err := s.LoadTests(courseID)
	if err != nil {
		return err
	}

	newTests := []models.Test{}
	found := false
	for _, t := range tests {
		if t.ID == testID {
			found = true
			continue
		}
		newTests = append(newTests, t)
	}

	if !found {
		return fmt.Errorf("test not found: %s", testID)
	}

	return s.SaveTests(courseID, newTests)
}

// Default feedback template embedded in binary
const defaultFeedbackTemplate = `Klasse:
Name:
Maximale Punkte:
Erreichte Punkte:
Note:

Feedback:
A1:
A2:
A3:
A4:
A5:
A6:
A7:
A8:
A9:
A10:
A11:
`

// ExportFeedbackFiles generates feedback.txt files for each student based on template
func (s *Storage) ExportFeedbackFiles(test *models.Test, course models.Course, outputDir string) error {
	// Use embedded template
	template := defaultFeedbackTemplate

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Calculate max points
	var maxPoints float64
	for _, q := range test.Questions {
		maxPoints += q.MaxPoints
	}
	maxPoints += test.GiftedPoints

	// Generate a file for each student
	for _, studentScore := range test.StudentScores {
		// Find student email from course
		var studentEmail string
		normalizedScoreName := normalizeString(studentScore.StudentName)
		for _, student := range course.Students {
			normalizedStudentName := normalizeString(student.Name)
			if strings.Contains(normalizedScoreName, normalizedStudentName) ||
				strings.Contains(normalizedStudentName, normalizedScoreName) {
				studentEmail = student.Email
				break
			}
		}

		// Skip if no email found - can't create proper filename
		if studentEmail == "" {
			continue
		}
		// Build feedback content
		content := template
		content = strings.Replace(content, "Klasse:", fmt.Sprintf("Klasse: %s", test.CourseName), 1)
		content = strings.Replace(content, "Name:", fmt.Sprintf("Name: %s", studentScore.StudentName), 1)
		content = strings.Replace(content, "Maximale Punkte:", fmt.Sprintf("Maximale Punkte: %.1f", maxPoints), 1)
		content = strings.Replace(content, "Erreichte Punkte:", fmt.Sprintf("Erreichte Punkte: %.1f", studentScore.TotalPoints), 1)
		content = strings.Replace(content, "Note:", fmt.Sprintf("Note: %.1f", studentScore.Grade), 1)

		// Replace task feedback (A1, A2, etc.)
		for i, question := range test.Questions {
			taskNum := i + 1
			taskKey := fmt.Sprintf("A%d:", taskNum)

			points := studentScore.QuestionScores[question.ID]
			comment := studentScore.QuestionComments[question.ID]

			// New multi-line format
			feedbackBlock := fmt.Sprintf("## A%d\nPunkte: %.1f/%.1f", taskNum, points, question.MaxPoints)
			if comment != "" {
				feedbackBlock += fmt.Sprintf("\nFeedback: %s", comment)
			} else {
				feedbackBlock += "\nFeedback:"
			}

			content = strings.Replace(content, taskKey, feedbackBlock, 1)
		}

		// Generate filename from email: get part before @, replace dots with dashes
		emailPrefix := strings.Split(studentEmail, "@")[0]
		emailPrefix = strings.ReplaceAll(emailPrefix, ".", "-")
		filename := fmt.Sprintf("%sfeedback.txt", emailPrefix)
		filepath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write feedback file for %s: %w", studentScore.StudentName, err)
		}
	}

	return nil
}

// normalizeString removes spaces, dashes, underscores and converts to lowercase for matching
func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}
