package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"cbrateach/internal/models"
)

// Structures to match the JSON input
type JSONImport struct {
	ExamName string                 `json:"exam_name"`
	Parts    map[string]JSONPart    `json:"parts"`
	Students map[string]JSONStudent `json:"students"`
}

type JSONPart struct {
	PartName string   `json:"part_name"`
	Tasks    []string `json:"tasks"`
}

type JSONStudent struct {
	Key     string                 `json:"Key"`
	Name    string                 `json:"name"`
	Surname string                 `json:"surname"`
	Results map[string]JSONPartRes `json:"results"`
}

type JSONPartRes map[string]JSONTaskRes

type JSONTaskRes struct {
	PointsReached  float64 `json:"points_reached"`
	PointsEarnable float64 `json:"points_earnable"`
	Comments       string  `json:"comments"`
	Reviewed       bool    `json:"reviewed"`
}

type MatchCandidate struct {
	OriginalName string
	Tokens       []string
}

// ParseTestJSON reads and parses the JSON file
func (s *Storage) ParseTestJSON(jsonPath string) (*JSONImport, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var importData JSONImport
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &importData, nil
}

// MatchStudents attempts to match JSON students to course students
// Returns:
// - matches: map of jsonKey -> studentName (for matched students)
// - unmatched: list of jsonKeys that couldn't be automatically matched
func (s *Storage) MatchStudents(importData *JSONImport, courseStudents []models.Student) (map[string]string, []string) {
	matches := make(map[string]string)
	var unmatched []string

	// Pre-process course students for matching
	var candidates []MatchCandidate
	for _, s := range courseStudents {
		tokens := tokenizeName(s.Name)
		candidates = append(candidates, MatchCandidate{
			OriginalName: s.Name,
			Tokens:       tokens,
		})
	}

	for _, jsonStudent := range importData.Students {
		// Fuzzy match name
		matchedName := findBestMatch(jsonStudent.Name, candidates)

		if matchedName != "" {
			matches[jsonStudent.Key] = matchedName
		} else {
			unmatched = append(unmatched, jsonStudent.Key)
		}
	}

	return matches, unmatched
}

// CreateTestFromJSON creates a Test model from import data and matches
func (s *Storage) CreateTestFromJSON(importData *JSONImport, matches map[string]string, courseID, courseName, testName, testTopic string, weight float64) (*models.Test, error) {
	// Extract Questions
	var partKeys []string
	for k := range importData.Parts {
		partKeys = append(partKeys, k)
	}
	sort.Strings(partKeys)

	var questions []models.Question
	questionMap := make(map[string]models.Question) // ID -> Question

	// To find max points, we need to look across all students or find one that has the task
	// Ideally max points should be consistent. We'll scan all students to find the max points for each task.
	// This is safer than picking just one student.

	taskMaxPoints := make(map[string]float64)
	for _, student := range importData.Students {
		for _, partRes := range student.Results {
			for taskKey, taskRes := range partRes {
				if taskRes.PointsEarnable > taskMaxPoints[taskKey] {
					taskMaxPoints[taskKey] = taskRes.PointsEarnable
				}
			}
		}
	}

	singlePart := len(partKeys) == 1

	for _, partKey := range partKeys {
		part := importData.Parts[partKey]
		for _, taskKey := range part.Tasks {
			maxPoints := taskMaxPoints[taskKey]

			// Naming: If single part, use taskKey (simplified).
			// If multiple parts, maybe prepend part name? user requested simplified for single part.
			// Currently we typically just use the taskKey as Title.
			title := taskKey
			// Potentially format it nicely: "task_1" -> "Task 1" ?
			// User asked "tasks are just called task_1 instead of part_1_task_1".
			// In the JSON, keys are "task_1".
			// If we were prefixing before locally, we stop. My previous code just used `taskKey`.
			// So `taskKey` is "task_1".
			// To be safe, if we have multiple parts, we might want to disambiguate if keys collide?
			// The JSON structure has tasks nested in parts. "Tasks": ["task_1", ...]
			// If different parts have same task keys, we definitely need prefix.

			if !singlePart {
				// Check for collision or just style?
				// For safety, let's prefix if multiple parts, to match user implication.
				title = fmt.Sprintf("%s_%s", part.PartName, taskKey)
			}

			q := models.Question{
				ID:        title, // Use title as ID for consistency/simplicity in this app
				Title:     title,
				MaxPoints: maxPoints,
			}
			questions = append(questions, q)
			questionMap[taskKey] = q // Map internal JSON task key to our Question
		}
	}

	// Process Students
	var studentScores []models.StudentScore

	// We process all students in the Import Data.
	// Matched ones get their real name.
	// Unmatched ones get their JSON name (Key or Name).

	// Wait, we need to know which students from the COURSE are missing?
	// The `Test` object has `StudentScores` which is a list.
	// Usually we populate this for ALL students in the class?
	// Or only for those we have scores for?
	// The TUI shows "Missing Students", so likely we only add those we have.

	// RE-DOING Question Generation Logic properly inside the score loop is inefficient.
	// Let's build a lookup map (PartName, TaskKey) -> QuestionID

	// ... (logic in next block) ...

	// Refined Question Mapping
	// Let's rebuild the question generation to allow lookup
	questionLookup := make(map[string]map[string]string) // PartKey -> TaskKey -> QuestionID

	questions = []models.Question{} // Reset

	for _, partKey := range partKeys {
		part := importData.Parts[partKey]
		questionLookup[partKey] = make(map[string]string)

		for _, taskKey := range part.Tasks {
			maxPoints := taskMaxPoints[taskKey]

			qID := taskKey
			if !singlePart {
				// Use PartKey or PartName? JSON has Part Key and Part Name.
				// Example: "part_1": { "part_name": "Word Formatting..." }
				// Let's use PartKey for ID component to be safe/short? Or Name?
				// User complained about "part_1_task_1". That matches `partKey`_"task_1".
				// So if single part, we avoid that.
				qID = fmt.Sprintf("%s_%s", partKey, taskKey)
			}

			q := models.Question{
				ID:        qID,
				Title:     qID,
				MaxPoints: maxPoints,
			}
			questions = append(questions, q)
			questionLookup[partKey][taskKey] = qID
		}
	}

	for _, jsonStudent := range importData.Students {
		finalName := jsonStudent.Name
		if matchedName, ok := matches[jsonStudent.Key]; ok {
			finalName = matchedName
		} else {
			finalName = fmt.Sprintf("%s (Ext)", jsonStudent.Name)
		}

		qScores := make(map[string]float64)
		qComments := make(map[string]string)

		for partKey, partRes := range jsonStudent.Results {
			for taskKey, taskRes := range partRes {
				// Look up mapped ID
				if partMap, ok := questionLookup[partKey]; ok {
					if qID, ok := partMap[taskKey]; ok {
						qScores[qID] = taskRes.PointsReached
						qComments[qID] = taskRes.Comments
					}
				}
			}
		}

		studentScore := models.StudentScore{
			StudentName:      finalName,
			QuestionScores:   qScores,
			QuestionComments: qComments,
		}

		studentScores = append(studentScores, studentScore)
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

	return &test, nil
}

// ImportTestFromJSON maintains backward compatibility for CLI
func (s *Storage) ImportTestFromJSON(jsonPath, courseID, courseName, testName, testTopic string, weight float64) error {
	importData, err := s.ParseTestJSON(jsonPath)
	if err != nil {
		return err
	}

	// Load course students
	courses, err := s.LoadCourses()
	if err != nil {
		return fmt.Errorf("failed to load courses: %w", err)
	}

	var courseStudents []models.Student
	for _, c := range courses {
		if c.ID == courseID {
			courseStudents = c.Students
			break
		}
	}

	if len(courseStudents) == 0 {
		return fmt.Errorf("course not found or has no students")
	}

	// Auto-match
	matches, unmatched := s.MatchStudents(importData, courseStudents)

	if len(unmatched) > 0 {
		fmt.Printf("Warning: %d students could not be matched automatically.\n", len(unmatched))
		for _, u := range unmatched {
			fmt.Printf(" - %s\n", importData.Students[u].Name)
		}
	}

	// Create Test
	test, err := s.CreateTestFromJSON(importData, matches, courseID, courseName, testName, testTopic, weight)
	if err != nil {
		return err
	}

	// Calculate grades
	s.RecalculateTestGrades(test)

	// Save test
	if err := s.AddTest(*test); err != nil {
		return fmt.Errorf("failed to save test: %w", err)
	}

	fmt.Printf("Successfully imported test '%s' from JSON for course '%s'\n", testName, courseName)
	fmt.Printf("  %d questions, %d students\n", len(test.Questions), len(test.StudentScores))

	return nil
}

func tokenizeName(name string) []string {
	parts := strings.Fields(strings.ToLower(name))
	return parts
}

func findBestMatch(inputName string, candidates []MatchCandidate) string {
	inputTokens := tokenizeName(inputName)

	bestMatch := ""
	bestScore := 0.0

	for _, cand := range candidates {
		matches := 0
		for _, cToken := range cand.Tokens {
			for _, iToken := range inputTokens {
				// simple match
				if iToken == cToken {
					matches++
					break
				}
			}
		}

		if len(cand.Tokens) == 0 {
			continue
		}

		score := float64(matches) / float64(len(cand.Tokens))

		if score > 0.5 && score > bestScore {
			bestScore = score
			bestMatch = cand.OriginalName
		}
	}

	return bestMatch
}
