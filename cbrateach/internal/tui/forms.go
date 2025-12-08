package tui

import (
	"errors"
	"os"

	"cbrateach/internal/models"

	"github.com/charmbracelet/huh"
)

type CourseFormResult struct {
	Name         string
	Subject      string
	Weekday      string
	Time         string
	Room         string
	CurrentTopic string
}

func ShowCourseForm() (*CourseFormResult, error) {
	result := &CourseFormResult{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Course Name").
				Value(&result.Name).
				Placeholder("e.g., Math 101").
				Validate(func(s string) error {
					if s == "" {
						return errors.New("course name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Subject").
				Value(&result.Subject).
				Placeholder("e.g., Mathematics"),

			huh.NewSelect[string]().
				Title("Weekday").
				Options(
					huh.NewOption("Monday", "Monday"),
					huh.NewOption("Tuesday", "Tuesday"),
					huh.NewOption("Wednesday", "Wednesday"),
					huh.NewOption("Thursday", "Thursday"),
					huh.NewOption("Friday", "Friday"),
					huh.NewOption("Saturday", "Saturday"),
					huh.NewOption("Sunday", "Sunday"),
				).
				Value(&result.Weekday),

			huh.NewInput().
				Title("Time").
				Value(&result.Time).
				Placeholder("e.g., 09:00"),

			huh.NewInput().
				Title("Room").
				Value(&result.Room).
				Placeholder("e.g., A-101"),

			huh.NewText().
				Title("Current Topic").
				Value(&result.CurrentTopic).
				Placeholder("What topic are you currently teaching?").
				Lines(3),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

type StudentFormResult struct {
	Name  string
	Email string
	Note  string
}

func ShowStudentForm() (*StudentFormResult, error) {
	result := &StudentFormResult{}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Student Name").
				Value(&result.Name).
				Placeholder("e.g., John Doe").
				Validate(func(s string) error {
					if s == "" {
						return errors.New("student name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Email").
				Value(&result.Email).
				Placeholder("e.g., john.doe@example.com"),

			huh.NewText().
				Title("Note (optional)").
				Value(&result.Note).
				Placeholder("Any notes about this student?").
				Lines(3),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

func ShowEditNoteForm(currentNote string) (string, error) {
	var note string = currentNote

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Edit Student Note").
				Value(&note).
				Lines(10),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return note, nil
}

type CourseEditFormResult struct {
	Subject      string
	Weekday      string
	Time         string
	Room         string
	CurrentTopic string
}

func ShowCourseEditForm(course *models.Course) (*CourseEditFormResult, error) {
	// Initialize with current values
	result := &CourseEditFormResult{
		Subject:      course.Subject,
		Weekday:      course.Weekday,
		Time:         course.Time,
		Room:         course.Room,
		CurrentTopic: course.CurrentTopic,
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Subject").
				Value(&result.Subject).
				Placeholder("e.g., Mathematics"),

			huh.NewSelect[string]().
				Title("Weekday").
				Options(
					huh.NewOption("Monday", "Monday"),
					huh.NewOption("Tuesday", "Tuesday"),
					huh.NewOption("Wednesday", "Wednesday"),
					huh.NewOption("Thursday", "Thursday"),
					huh.NewOption("Friday", "Friday"),
					huh.NewOption("Saturday", "Saturday"),
					huh.NewOption("Sunday", "Sunday"),
				).
				Value(&result.Weekday),

			huh.NewInput().
				Title("Time").
				Value(&result.Time).
				Placeholder("e.g., 09:00"),

			huh.NewInput().
				Title("Room").
				Value(&result.Room).
				Placeholder("e.g., A-101"),

			huh.NewText().
				Title("Current Topic").
				Value(&result.CurrentTopic).
				Placeholder("What topic are you currently teaching?").
				Lines(3),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

func ShowExportFormatChoice() (string, error) {
	var format string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose export format").
				Options(
					huh.NewOption("CSV", "csv"),
					huh.NewOption("Excel (XLSX)", "xlsx"),
				).
				Value(&format),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return format, nil
}

func ShowMessage(title, message string) error {
	var ok bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(title).
				Description(message),
			huh.NewConfirm().
				Title("").
				Affirmative("OK").
				Value(&ok),
		),
	)

	return form.Run()
}

type FeedbackFormResult struct {
	FeedbackDir   string
	CustomMessage string
}

func ShowFeedbackForm() (*FeedbackFormResult, error) {
	result := &FeedbackFormResult{}

	// Get home directory for default
	homeDir, _ := os.UserHomeDir()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Feedback Files Directory").
				Description("Directory containing feedback files (named with student names)").
				Value(&result.FeedbackDir).
				Placeholder(homeDir).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("directory is required")
					}
					// Check if directory exists
					if _, err := os.Stat(s); os.IsNotExist(err) {
						return errors.New("directory does not exist")
					}
					return nil
				}),

			huh.NewText().
				Title("Custom Message (optional)").
				Description("Will replace {{CustomMessage}} in template").
				Value(&result.CustomMessage).
				Lines(5),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return result, nil
}

func ShowConfirmation(title, message, confirmLabel, cancelLabel string) (bool, error) {
	var confirmed bool

	// Default labels if empty
	if confirmLabel == "" {
		confirmLabel = "Yes"
	}
	if cancelLabel == "" {
		cancelLabel = "No"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title(title).
				Description(message),
			huh.NewConfirm().
				Title("").
				Affirmative(confirmLabel).
				Negative(cancelLabel).
				Value(&confirmed),
		),
	)

	if err := form.Run(); err != nil {
		return false, err
	}

	return confirmed, nil
}
