package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cbrateach/internal/config"
	"cbrateach/internal/models"
)

type Storage struct {
	cfg config.Config
}

func New(cfg config.Config) *Storage {
	return &Storage{cfg: cfg}
}

// Courses

func (s *Storage) CoursesPath() string {
	return filepath.Join(s.cfg.DataDir, "courses.json")
}

func (s *Storage) LoadCourses() ([]models.Course, error) {
	path := s.CoursesPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Course{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var courses []models.Course
	if err := json.Unmarshal(data, &courses); err != nil {
		return nil, err
	}

	return courses, nil
}

func (s *Storage) SaveCourses(courses []models.Course) error {
	data, err := json.MarshalIndent(courses, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.CoursesPath(), data, 0644)
}

// Reviews

func (s *Storage) SaveReview(review models.Review) error {
	// Save review as JSON
	filename := fmt.Sprintf("%s_%s.json",
		review.Date.Format("2006-01-02"),
		sanitizeFilename(review.CourseName))
	path := filepath.Join(s.cfg.ReviewsDir, filename)

	data, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	// If review has text, append to course note MD file
	if review.ReviewText != "" {
		return s.AppendReviewToNote(review)
	}

	return nil
}

func (s *Storage) AppendReviewToNote(review models.Review) error {
	// Find the course to get its note file
	courses, err := s.LoadCourses()
	if err != nil {
		return err
	}

	var course *models.Course
	for i := range courses {
		if courses[i].ID == review.CourseID {
			course = &courses[i]
			break
		}
	}

	if course == nil {
		return fmt.Errorf("course not found: %s", review.CourseID)
	}

	notePath := filepath.Join(s.cfg.CourseNotesDir, course.NoteFile)

	// Read existing note
	var content string
	if data, err := os.ReadFile(notePath); err == nil {
		content = string(data)
	}

	// Check if Reviews section exists
	reviewsSection := "### Reviews"
	if !strings.Contains(content, reviewsSection) {
		// Add Reviews section if it doesn't exist
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + reviewsSection + "\n\n"
	}

	// Append the new review
	reviewEntry := fmt.Sprintf("\n**%s** - %s\n\n%s\n",
		review.Date.Format("2006-01-02"),
		review.Topic,
		review.ReviewText)

	content += reviewEntry

	return os.WriteFile(notePath, []byte(content), 0644)
}

func (s *Storage) LoadReviews() ([]models.Review, error) {
	files, err := os.ReadDir(s.cfg.ReviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Review{}, nil
		}
		return nil, err
	}

	var reviews []models.Review
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.cfg.ReviewsDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var review models.Review
		if err := json.Unmarshal(data, &review); err != nil {
			continue
		}

		reviews = append(reviews, review)
	}

	return reviews, nil
}

// Course notes

func (s *Storage) CreateCourseNote(course *models.Course) error {
	if course.NoteFile == "" {
		course.NoteFile = sanitizeFilename(course.Name) + ".md"
	}

	notePath := filepath.Join(s.cfg.CourseNotesDir, course.NoteFile)

	// Create initial note with course info
	content := fmt.Sprintf("# %s\n\n**Subject:** %s\n**Time:** %s %s\n**Room:** %s\n\n## Current Topic\n\n%s\n\n### Reviews\n\n",
		course.Name,
		course.Subject,
		course.Weekday,
		course.Time,
		course.Room,
		course.CurrentTopic)

	return os.WriteFile(notePath, []byte(content), 0644)
}

// Utility functions

func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func GenerateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
