package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cbrateach/internal/models"

	"github.com/xuri/excelize/v2"
)

// ImportStudentsFromCSV imports students from a CSV or XLSX file into the specified course
// Format expected: name,email (with optional header row)
func (s *Storage) ImportStudentsFromCSV(filePath, courseName string) error {
	// Detect file type by extension
	ext := strings.ToLower(filepath.Ext(filePath))

	var records [][]string
	var err error

	switch ext {
	case ".csv":
		records, err = readCSV(filePath)
	case ".xlsx", ".xls":
		records, err = readXLSX(filePath)
	default:
		return fmt.Errorf("unsupported file type: %s (supported: .csv, .xlsx)", ext)
	}

	if err != nil {
		return err
	}

	if len(records) == 0 {
		return fmt.Errorf("file is empty")
	}

	// Load existing courses
	courses, err := s.LoadCourses()
	if err != nil {
		return fmt.Errorf("failed to load courses: %w", err)
	}

	// Find the target course
	var courseIdx = -1
	for i, course := range courses {
		if course.Name == courseName {
			courseIdx = i
			break
		}
	}

	if courseIdx == -1 {
		return fmt.Errorf("course not found: %s", courseName)
	}

	// Parse students from CSV
	startRow := 0

	// Check if first row is a header (contains "name" or "email")
	if len(records) > 0 && len(records[0]) >= 2 {
		firstRow := records[0]
		if firstRow[0] == "name" || firstRow[0] == "Name" ||
		   firstRow[1] == "email" || firstRow[1] == "Email" {
			startRow = 1
		}
	}

	// Import students
	imported := 0
	for i := startRow; i < len(records); i++ {
		record := records[i]

		if len(record) < 2 {
			continue // Skip incomplete rows
		}

		name := record[0]
		email := record[1]

		// Skip empty rows
		if name == "" {
			continue
		}

		// Check if student already exists
		exists := false
		for _, student := range courses[courseIdx].Students {
			if student.Name == name {
				exists = true
				break
			}
		}

		if !exists {
			student := models.Student{
				Name:  name,
				Email: email,
			}
			courses[courseIdx].Students = append(courses[courseIdx].Students, student)
			imported++
		}
	}

	// Save updated courses
	if err := s.SaveCourses(courses); err != nil {
		return fmt.Errorf("failed to save courses: %w", err)
	}

	fmt.Printf("Successfully imported %d students into course '%s'\n", imported, courseName)
	return nil
}

// readCSV reads a CSV file and returns rows as [][]string
func readCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	return records, nil
}

// readXLSX reads an Excel file and returns rows from the first sheet as [][]string
func readXLSX(filePath string) ([][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer f.Close()

	// Get the first sheet name
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("XLSX file has no sheets")
	}

	// Read all rows from the first sheet
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("failed to read XLSX rows: %w", err)
	}

	return rows, nil
}

// ImportCourseFromSchoolXLSX imports a course and students from school-specific XLSX format
// Format: Row 1 = "Klasse <name>", Row 3 = headers, Row 4+ = Vorname, Nachname, Email
func (s *Storage) ImportCourseFromSchoolXLSX(filePath string) error {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to open XLSX file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return fmt.Errorf("XLSX file has no sheets")
	}

	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return fmt.Errorf("failed to read XLSX rows: %w", err)
	}

	if len(rows) < 4 {
		return fmt.Errorf("file doesn't have enough rows (expected at least 4)")
	}

	// Extract course name from row 1
	// Format: "Klasse 29Gc" -> "29Gc"
	var courseName string
	if len(rows[0]) > 0 {
		courseNameFull := rows[0][0]
		// Remove "Klasse " prefix if present
		if len(courseNameFull) > 7 && courseNameFull[:7] == "Klasse " {
			courseName = courseNameFull[7:]
		} else {
			courseName = courseNameFull
		}
	}

	if courseName == "" {
		courseName = sheetName // Fallback to sheet name
	}

	// Load existing courses
	courses, err := s.LoadCourses()
	if err != nil {
		return fmt.Errorf("failed to load courses: %w", err)
	}

	// Check if course already exists
	var courseIdx = -1
	for i, course := range courses {
		if course.Name == courseName {
			courseIdx = i
			break
		}
	}

	// If course doesn't exist, create it
	if courseIdx == -1 {
		newCourse := models.Course{
			ID:       GenerateID(),
			Name:     courseName,
			Subject:  "", // User can fill this in later
			Students: []models.Student{},
		}

		// Create note file for the course
		if err := s.CreateCourseNote(&newCourse); err != nil {
			return fmt.Errorf("failed to create course note: %w", err)
		}

		courses = append(courses, newCourse)
		courseIdx = len(courses) - 1

		fmt.Printf("Created new course: %s\n", courseName)
	}

	// Import students (starting from row 4, index 3)
	imported := 0
	for i := 3; i < len(rows); i++ {
		row := rows[i]

		// Need at least 3 columns: Vorname, Nachname, Email
		if len(row) < 3 {
			continue
		}

		vorname := row[0]
		nachname := row[1]
		email := row[2]

		// Skip empty rows
		if vorname == "" && nachname == "" {
			continue
		}

		// Combine first and last name
		fullName := fmt.Sprintf("%s %s", vorname, nachname)

		// Check if student already exists
		exists := false
		for _, student := range courses[courseIdx].Students {
			if student.Name == fullName {
				exists = true
				break
			}
		}

		if !exists {
			student := models.Student{
				Name:  fullName,
				Email: email,
			}
			courses[courseIdx].Students = append(courses[courseIdx].Students, student)
			imported++
		}
	}

	// Save updated courses
	if err := s.SaveCourses(courses); err != nil {
		return fmt.Errorf("failed to save courses: %w", err)
	}

	fmt.Printf("Successfully imported %d students into course '%s'\n", imported, courseName)
	return nil
}
