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
