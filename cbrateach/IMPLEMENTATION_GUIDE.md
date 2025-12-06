# UI Improvements Implementation Guide

This guide details how to implement the remaining UI features. All backend functionality (weights, weighted averages) is already complete.

## 1. Test Review View Enhancements

### 1.1 Show Missing Students Below Table

**File:** `internal/tui/test_review_view.go`

**Location:** In `renderTestReviewView()`, after the table and before help text

**Implementation:**
```go
// After the table.View() call, add:

// Find missing students
course := m.courses[m.selectedCourse]
missingStudents := []string{}
for _, student := range course.Students {
    found := false
    for _, score := range test.StudentScores {
        if score.StudentName == student.Name {
            found = true
            break
        }
    }
    if !found {
        missingStudents = append(missingStudents, student.Name)
    }
}

if len(missingStudents) > 0 {
    b.WriteString("\n")
    b.WriteString(subtitleStyle.Render("Missing Students:") + "\n")
    for _, name := range missingStudents {
        b.WriteString(fmt.Sprintf("  • %s\n", name))
    }
    b.WriteString(helpStyle.Render("Press 'm' to add missing student") + "\n")
}
```

**In `updateTestReviewView()`**, add key handler:
```go
case "m":
    // Add missing student to test
    if test.Status == "review" && len(missingStudents) > 0 {
        return m, m.addMissingStudentToTest()
    }
```

**New function:**
```go
func (m Model) addMissingStudentToTest() tea.Cmd {
    return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
        // Use huh to select which missing student to add
        // Then add them with zero scores for all questions
        return nil
    })
}
```

### 1.2 Highlight Editing Cell in Different Color

**File:** `internal/tui/test_review_view.go`

**Location:** In the row building loop

**Current code:**
```go
if m.editingCell && m.selectedRow == i && m.selectedCol == j {
    cellValue = fmt.Sprintf("[%s_]", m.editValue)
}
```

**Enhanced version:**
```go
if m.editingCell && m.selectedRow == i && m.selectedCol == j {
    // Highlight in yellow/orange
    editStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#000")).
        Background(lipgloss.Color("#FFA500")).
        Bold(true)
    cellValue = editStyle.Render(fmt.Sprintf("%s_", m.editValue))
}
```

### 1.3 Vertical Bar Chart for Grade Distribution

**Files:**
- Add to `internal/tui/test_review_view.go`
- May need: `go get github.com/NimbleMarkets/ntcharts/barchart`

**Alternative (simple text-based):**
```go
func (m Model) renderGradeDistribution(test models.Test) string {
    // Count grades
    distribution := make(map[float64]int)
    for _, score := range test.StudentScores {
        // Round to nearest 0.25
        grade := score.Grade
        distribution[grade]++
    }

    var b strings.Builder
    b.WriteString(subtitleStyle.Render("Grade Distribution") + "\n\n")

    // Create vertical bars
    maxCount := 0
    for _, count := range distribution {
        if count > maxCount {
            maxCount = count
        }
    }

    // Vertical bar chart (simple version)
    grades := []float64{1.0, 1.25, 1.5, 1.75, 2.0, 2.25, 2.5, 2.75, 3.0, 3.25, 3.5, 3.75, 4.0, 4.25, 4.5, 4.75, 5.0, 5.25, 5.5, 5.75, 6.0}

    // Print bars from top to bottom
    for height := maxCount; height > 0; height-- {
        for _, grade := range grades {
            count := distribution[grade]
            if count >= height {
                b.WriteString("█ ")
            } else {
                b.WriteString("  ")
            }
        }
        b.WriteString(fmt.Sprintf(" %d\n", height))
    }

    // Print grade labels
    for _, grade := range grades {
        if int(grade*4)%4 == 0 { // Show only whole numbers
            b.WriteString(fmt.Sprintf("%.0f ", grade))
        } else {
            b.WriteString("  ")
        }
    }

    return b.String()
}
```

**Add to `renderTestReviewView()`** after statistics, before help text.

## 2. Course View Enhancements

### 2.1 Editable Course Details

**File:** `internal/tui/classbook_view.go`

**Add key handler in `updateClassbookView()`:**
```go
case "d":
    // Edit course details
    return m, m.editCourseDetails()
```

**New function:**
```go
func (m Model) editCourseDetails() tea.Cmd {
    return tea.ExecProcess(exec.Command("true"), func(err error) tea.Msg {
        if m.selectedCourse >= len(m.courses) {
            return nil
        }

        course := &m.courses[m.selectedCourse]

        // Create form with current values
        form := huh.NewForm(
            huh.NewGroup(
                huh.NewInput().
                    Title("Subject").
                    Value(&course.Subject).
                    Placeholder("e.g., Mathematics"),

                huh.NewSelect[string]().
                    Title("Weekday").
                    Options(
                        huh.NewOption("Monday", "Monday"),
                        huh.NewOption("Tuesday", "Tuesday"),
                        huh.NewOption("Wednesday", "Wednesday"),
                        huh.NewOption("Thursday", "Thursday"),
                        huh.NewOption("Friday", "Friday"),
                    ).
                    Value(&course.Weekday),

                huh.NewInput().
                    Title("Time").
                    Value(&course.Time).
                    Placeholder("e.g., 09:00"),

                huh.NewInput().
                    Title("Room").
                    Value(&course.Room).
                    Placeholder("e.g., A-101"),

                huh.NewText().
                    Title("Current Topic").
                    Value(&course.CurrentTopic).
                    Lines(3),
            ),
        )

        if err := form.Run(); err != nil {
            return nil
        }

        // Save changes
        m.storage.SaveCourses(m.courses)

        return nil
    })
}
```

**Update help text** to include `d: edit details`

### 2.2 Display Topic Tag in Course List

**File:** `internal/tui/list_view.go`

**Location:** In `renderListView()`, in the course list loop

**Current:**
```go
line := fmt.Sprintf("%s %s - %s (%s %s, Room %s)",
    cursor,
    course.Name,
    course.Subject,
    course.Weekday,
    course.Time,
    course.Room)
```

**Enhanced:**
```go
line := fmt.Sprintf("%s %s - %s (%s %s, Room %s)",
    cursor,
    course.Name,
    course.Subject,
    course.Weekday,
    course.Time,
    course.Room)

// Add topic tag
if course.CurrentTopic != "" {
    topicStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFF")).
        Background(primaryColor).
        Padding(0, 1).
        MarginLeft(1)

    line += " " + topicStyle.Render(course.CurrentTopic)
}
```

### 2.3 Test Shortcut in List View

**File:** `internal/tui/list_view.go`

**Add in `updateListView()`:**
```go
case "t":
    // Open tests for selected course
    if len(m.courses) > 0 && m.cursor < len(m.courses) {
        m.selectedCourse = m.cursor
        course := m.courses[m.cursor]
        tests, _ := m.storage.LoadTests(course.ID)
        m.tests = tests
        m.cursor = 0
        m.state = testListView
    }
```

**Update help text** to include `t: tests`

## 3. Student View - Display Grades

**File:** `internal/tui/classbook_view.go`

**Location:** In `renderCourseDetails()`, in the "Selected Student" section

**Add after student info:**
```go
// Load and display student's test grades
tests, err := m.storage.LoadTests(course.ID)
if err == nil && len(tests) > 0 {
    b.WriteString("\n" + subtitleStyle.Render("Test Grades:") + "\n")

    var totalWeightedGrade float64
    var totalWeight float64

    for _, test := range tests {
        // Find this student's score in the test
        for _, score := range test.StudentScores {
            if score.StudentName == student.Name {
                weight := test.Weight
                if weight <= 0 {
                    weight = 1.0
                }

                statusIcon := "📝"
                if test.Status == "confirmed" {
                    statusIcon = "✓"
                    totalWeightedGrade += score.Grade * weight
                    totalWeight += weight
                }

                b.WriteString(fmt.Sprintf("  %s %s: %.2f (weight: %.1f)\n",
                    statusIcon, test.Title, score.Grade, weight))
                break
            }
        }
    }

    // Calculate and show average
    if totalWeight > 0 {
        avgGrade := totalWeightedGrade / totalWeight
        b.WriteString("\n")
        avgStyle := lipgloss.NewStyle().
            Foreground(successColor).
            Bold(true)
        b.WriteString(avgStyle.Render(fmt.Sprintf("Average Grade: %.2f", avgGrade)) + "\n")
    }
}
```

## 4. Review Form Refactoring

**File:** `internal/tui/review_form.go`

**Replace `ShowReviewForm()` with:**

```go
func ShowReviewForm(course models.Course) (*ReviewFormResult, error) {
    // Default values
    defaultTitle := fmt.Sprintf("%s - %s", course.Name, time.Now().Format("2006-01-02"))
    defaultTopic := course.CurrentTopic

    result := &ReviewFormResult{}

    // First part: basic info
    basicForm := huh.NewForm(
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
    )

    if err := basicForm.Run(); err != nil {
        return nil, err
    }

    // Set defaults if not provided
    if result.Title == "" {
        result.Title = defaultTitle
    }
    if result.Topic == "" {
        result.Topic = defaultTopic
    }

    // Second part: student selection with p/n toggles
    // Create a custom multi-select style interface
    type StudentMark struct {
        Name string
        Mark string // "", "p", or "n"
    }

    studentMarks := make([]StudentMark, len(course.Students))
    for i, student := range course.Students {
        studentMarks[i] = StudentMark{Name: student.Name, Mark: ""}
    }

    // Build the selection interface
    // Note: This requires a custom bubbletea component
    // For simplicity, you might want to use a simpler approach:

    // Option 1: Separate questions for positive and negative
    var positiveStudents []string
    var negativeStudents []string

    positiveOptions := make([]huh.Option[string], 0)
    for _, student := range course.Students {
        positiveOptions = append(positiveOptions,
            huh.NewOption(student.Name, student.Name))
    }

    selectForm := huh.NewForm(
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Students who stood out POSITIVELY (✓)").
                Description("Select all that apply").
                Options(positiveOptions...).
                Value(&positiveStudents).
                Height(10),
        ),
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Students who need attention (-)").
                Description("Select all that apply").
                Options(positiveOptions...).
                Value(&negativeStudents).
                Height(10),
        ),
    )

    if err := selectForm.Run(); err != nil {
        return nil, err
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

    return result, nil
}
```

## 5. Styling Reference

**File:** `internal/tui/styles.go`

Add these if needed:

```go
var (
    // Additional colors for new features
    warningColor  = lipgloss.Color("#F59E0B")
    infoColor     = lipgloss.Color("#3B82F6")

    // Tag style for topics
    tagStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFF")).
        Background(primaryColor).
        Padding(0, 1).
        MarginLeft(1)

    // Editing cell style
    editCellStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#000")).
        Background(warningColor).
        Bold(true)
)
```

## 6. Testing Checklist

- [ ] Test weight flag: `./cbrateach add-test --test-weight 2.0 ...`
- [ ] Verify weighted export works correctly
- [ ] Test missing students display
- [ ] Test course detail editing
- [ ] Test topic tags display
- [ ] Test shortcut 't' from list view
- [ ] Test student grades display
- [ ] Test review form with new flow

## 7. Optional: Grade Distribution Chart with Library

If you want a fancier chart, install:
```bash
go get github.com/NimbleMarkets/ntcharts/barchart
```

Then use:
```go
import "github.com/NimbleMarkets/ntcharts/barchart"

func (m Model) renderGradeDistributionChart(test models.Test) string {
    bc := barchart.New(60, 15) // width, height

    // Prepare data
    distribution := make(map[float64]int)
    for _, score := range test.StudentScores {
        distribution[score.Grade]++
    }

    // Add bars
    for grade := 1.0; grade <= 6.0; grade += 0.25 {
        count := distribution[grade]
        bc.Push(barchart.BarData{
            Label: fmt.Sprintf("%.2f", grade),
            Value: count,
        })
    }

    return bc.View()
}
```

## Notes

- All data layer changes are complete
- UI changes are purely visual/interaction improvements
- You can implement these incrementally
- Test each feature before moving to the next
- The existing codebase has good patterns to follow

Good luck! The foundation is solid, these are all polish features.
