package tui

import (
	"fmt"
	"time"

	"cbrateach/internal/models"
	"cbrateach/internal/storage"

	"github.com/charmbracelet/huh"
)

type ReviewFormResult struct {
	Title      string
	Topic      string
	ReviewText string
	Students   []models.ReviewStudent
}

func ShowReviewForm(course models.Course) (*ReviewFormResult, error) {
	// Default values
	defaultTitle := fmt.Sprintf("%s - %s", course.Name, time.Now().Format("2006-01-02"))
	defaultTopic := course.CurrentTopic

	result := &ReviewFormResult{}

	// Student selection options
	studentOptions := make([]huh.Option[string], 0)
	for _, student := range course.Students {
		studentOptions = append(studentOptions,
			huh.NewOption(student.Name, student.Name))
	}

	var selectedStudents []string
	var positiveStudents []string
	var negativeStudents []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Review Title").
				Value(&result.Title).
				Placeholder(defaultTitle),

			huh.NewInput().
				Title("Topic Discussed").
				Value(&result.Topic).
				Placeholder(defaultTopic),

			huh.NewText().
				Title("Review Notes (optional)").
				Value(&result.ReviewText).
				Placeholder("What happened in this class?").
				Lines(5),
		),

		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Students who stood out (select all)").
				Options(studentOptions...).
				Value(&selectedStudents).
				Height(10),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Set defaults if not provided
	if result.Title == "" {
		result.Title = defaultTitle
	}
	if result.Topic == "" {
		result.Topic = defaultTopic
	}

	// If students were selected, ask about positive/negative
	if len(selectedStudents) > 0 {
		// Build options for selected students
		selectedOptions := make([]huh.Option[string], 0)
		for _, name := range selectedStudents {
			selectedOptions = append(selectedOptions,
				huh.NewOption(name, name))
		}

		// Ask which were positive
		positiveForm := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Who stood out POSITIVELY?").
					Options(selectedOptions...).
					Value(&positiveStudents).
					Height(10),
			),
		)

		if err := positiveForm.Run(); err != nil {
			return nil, err
		}

		// Remaining are negative
		positiveMap := make(map[string]bool)
		for _, name := range positiveStudents {
			positiveMap[name] = true
		}

		for _, name := range selectedStudents {
			if !positiveMap[name] {
				negativeStudents = append(negativeStudents, name)
			}
		}

		// Build result
		for _, name := range positiveStudents {
			result.Students = append(result.Students, models.ReviewStudent{
				Name:     name,
				Positive: true,
				Reason:   "Stood out positively",
			})
		}

		for _, name := range negativeStudents {
			result.Students = append(result.Students, models.ReviewStudent{
				Name:     name,
				Positive: false,
				Reason:   "Needs attention",
			})
		}
	}

	return result, nil
}

func SaveReview(store *storage.Storage, course models.Course, formResult *ReviewFormResult) error {
	review := models.Review{
		ID:              storage.GenerateID(),
		CourseID:        course.ID,
		CourseName:      course.Name,
		Date:            time.Now(),
		Title:           formResult.Title,
		Topic:           formResult.Topic,
		ReviewText:      formResult.ReviewText,
		StudentsStandOut: formResult.Students,
	}

	// Save the review
	if err := store.SaveReview(review); err != nil {
		return err
	}

	// Update student marks
	courses, err := store.LoadCourses()
	if err != nil {
		return err
	}

	for i := range courses {
		if courses[i].ID == course.ID {
			for _, rs := range formResult.Students {
				// Find student and add mark
				for j := range courses[i].Students {
					if courses[i].Students[j].Name == rs.Name {
						mark := models.Mark{
							Date:   time.Now(),
							Reason: rs.Reason,
						}

						if rs.Positive {
							courses[i].Students[j].PositiveMarks = append(
								courses[i].Students[j].PositiveMarks, mark)
						} else {
							courses[i].Students[j].NegativeMarks = append(
								courses[i].Students[j].NegativeMarks, mark)
						}
					}
				}
			}
			break
		}
	}

	return store.SaveCourses(courses)
}
