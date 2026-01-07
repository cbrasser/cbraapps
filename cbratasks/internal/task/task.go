package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	List        string     `json:"list"` // Task list name (e.g., "work", "personal", "inbox")
	DueDate     *time.Time `json:"due_date,omitempty"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Archived    bool       `json:"archived"`
	Href        string     `json:"href,omitempty"` // Remote resource path

	// Deprecated fields - kept for migration only
	Tags     []string `json:"tags,omitempty"`
	ListName string   `json:"list_name,omitempty"`
}

// NewTask creates a new task with the given title and list
func NewTask(title string, list string) *Task {
	now := time.Now()
	return &Task{
		ID:        uuid.New().String(),
		Title:     title,
		List:      list,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Complete marks a task as completed
func (t *Task) Complete() {
	now := time.Now()
	t.Completed = true
	t.CompletedAt = &now
	t.UpdatedAt = now
}

// Uncomplete marks a task as not completed
func (t *Task) Uncomplete() {
	t.Completed = false
	t.CompletedAt = nil
	t.UpdatedAt = time.Now()
}

// ToggleComplete toggles the completed status
func (t *Task) ToggleComplete() {
	if t.Completed {
		t.Uncomplete()
	} else {
		t.Complete()
	}
}

// SetList sets the task's list
func (t *Task) SetList(list string) {
	list = strings.ToLower(strings.TrimSpace(list))
	t.List = list
	t.UpdatedAt = time.Now()
}

// MigrateFromOldFormat converts old tag/ListName format to new list-based format
func (t *Task) MigrateFromOldFormat(defaultList string) {
	// If already has a list, no migration needed
	if t.List != "" {
		return
	}

	// Try migrating from old Tags field (use first tag as list)
	if len(t.Tags) > 0 {
		t.List = strings.ToLower(strings.TrimSpace(t.Tags[0]))
		t.Tags = nil // Clear tags after migration
		t.ListName = ""
		t.UpdatedAt = time.Now()
		return
	}

	// Otherwise use default list
	t.List = defaultList
	t.ListName = ""
	t.UpdatedAt = time.Now()
}

// SetDueDate sets the due date
func (t *Task) SetDueDate(d time.Time) {
	t.DueDate = &d
	t.UpdatedAt = time.Now()
}

// ShouldArchive returns true if the task should be archived
// (completed more than 24 hours ago)
func (t *Task) ShouldArchive() bool {
	if !t.Completed || t.CompletedAt == nil {
		return false
	}
	return time.Since(*t.CompletedAt) > 24*time.Hour
}

// IsOverdue returns true if the task is overdue
func (t *Task) IsOverdue() bool {
	if t.Completed || t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate)
}

// IsDueToday returns true if task is due today
func (t *Task) IsDueToday() bool {
	if t.DueDate == nil {
		return false
	}
	now := time.Now()
	due := *t.DueDate
	return due.Year() == now.Year() && due.YearDay() == now.YearDay()
}

// DueString returns a human-readable due date string
func (t *Task) DueString() string {
	if t.DueDate == nil {
		return ""
	}

	now := time.Now()
	due := *t.DueDate

	// Check if it's today
	if due.Year() == now.Year() && due.YearDay() == now.YearDay() {
		return "Today"
	}

	// Check if it's tomorrow
	tomorrow := now.AddDate(0, 0, 1)
	if due.Year() == tomorrow.Year() && due.YearDay() == tomorrow.YearDay() {
		return "Tomorrow"
	}

	// Within this week
	daysUntil := int(due.Sub(now).Hours() / 24)
	if daysUntil > 0 && daysUntil < 7 {
		return due.Format("Mon")
	}

	// Default format
	return due.Format("02 Jan")
}

// ParseDueDate parses various date formats into a time.Time
// Supports: +1d, +3d, +1w, +2w, tomorrow, nextweek, DD-MM-YYYY
func ParseDueDate(input string) (*time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	now := time.Now()

	// Relative dates: +1d, +3d, +1w, 1d, 3d, 1w etc. (with or without +)
	relativeRegex := regexp.MustCompile(`^\+?(\d+)([dwm])$`)
	if matches := relativeRegex.FindStringSubmatch(input); matches != nil {
		num, _ := strconv.Atoi(matches[1])
		unit := matches[2]

		var result time.Time
		switch unit {
		case "d":
			result = now.AddDate(0, 0, num)
		case "w":
			result = now.AddDate(0, 0, num*7)
		case "m":
			result = now.AddDate(0, num, 0)
		}
		// Set to end of day
		result = time.Date(result.Year(), result.Month(), result.Day(), 23, 59, 59, 0, result.Location())
		return &result, nil
	}

	// Keywords
	switch input {
	case "today":
		result := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		return &result, nil
	case "tomorrow":
		result := now.AddDate(0, 0, 1)
		result = time.Date(result.Year(), result.Month(), result.Day(), 23, 59, 59, 0, result.Location())
		return &result, nil
	case "nextweek":
		// Next Monday
		daysUntilMonday := (8 - int(now.Weekday())) % 7
		if daysUntilMonday == 0 {
			daysUntilMonday = 7
		}
		result := now.AddDate(0, 0, daysUntilMonday)
		result = time.Date(result.Year(), result.Month(), result.Day(), 23, 59, 59, 0, result.Location())
		return &result, nil
	}

	// Specific date: DD-MM-YYYY
	if t, err := time.Parse("02-01-2006", input); err == nil {
		t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, now.Location())
		return &t, nil
	}

	// Specific date: YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", input); err == nil {
		t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, now.Location())
		return &t, nil
	}

	return nil, fmt.Errorf("invalid date format: %s", input)
}

// ToJSON serializes the task to JSON
func (t *Task) ToJSON() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// FromJSON deserializes a task from JSON
func FromJSON(data []byte) (*Task, error) {
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
