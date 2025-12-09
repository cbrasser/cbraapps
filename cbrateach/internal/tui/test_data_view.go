package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateTestDataView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		// Go back to test review
		m.state = testReviewView
		return m, nil
	}

	return m, nil
}

func (m Model) renderTestDataView() string {
	if m.selectedTest >= len(m.tests) {
		m.state = testListView
		return m.renderTestListView()
	}

	test := m.tests[m.selectedTest]

	var b strings.Builder

	// Title
	titleText := fmt.Sprintf("Test Data: %s - %s", test.Title, test.Topic)
	title := titleStyle.Render(titleText)
	b.WriteString(title + "\n\n")

	// Calculate statistics
	avgGrade := 0.0
	minGrade := 6.0
	maxGrade := 1.0
	avgTotal := 0.0
	minTotal := 999999.0
	maxTotal := 0.0

	for _, score := range test.StudentScores {
		avgGrade += score.Grade
		if score.Grade < minGrade {
			minGrade = score.Grade
		}
		if score.Grade > maxGrade {
			maxGrade = score.Grade
		}

		avgTotal += score.TotalPoints
		if score.TotalPoints < minTotal {
			minTotal = score.TotalPoints
		}
		if score.TotalPoints > maxTotal {
			maxTotal = score.TotalPoints
		}
	}

	if len(test.StudentScores) > 0 {
		avgGrade /= float64(len(test.StudentScores))
		avgTotal /= float64(len(test.StudentScores))
	} else {
		minGrade = 0
		minTotal = 0
	}

	maxPoints := 0.0
	for _, q := range test.Questions {
		maxPoints += q.MaxPoints
	}

	// Overall Statistics
	b.WriteString(subtitleStyle.Render("Overall Statistics") + "\n\n")
	b.WriteString(fmt.Sprintf("  Students:        %d\n", len(test.StudentScores)))
	b.WriteString(fmt.Sprintf("  Max Points:      %.1f\n", maxPoints))
	b.WriteString(fmt.Sprintf("  Gifted Points:   %.1f\n", test.GiftedPoints))
	b.WriteString(fmt.Sprintf("  Weight:          %.1f\n\n", test.Weight))

	// Grade Statistics
	b.WriteString(subtitleStyle.Render("Grade Statistics") + "\n\n")
	b.WriteString(fmt.Sprintf("  Average Grade:   %.2f\n", avgGrade))
	b.WriteString(fmt.Sprintf("  Min Grade:       %.2f\n", minGrade))
	b.WriteString(fmt.Sprintf("  Max Grade:       %.2f\n\n", maxGrade))

	// Points Statistics
	b.WriteString(subtitleStyle.Render("Points Statistics") + "\n\n")
	b.WriteString(fmt.Sprintf("  Average Points:  %.1f\n", avgTotal))
	b.WriteString(fmt.Sprintf("  Min Points:      %.1f\n", minTotal))
	b.WriteString(fmt.Sprintf("  Max Points:      %.1f\n\n", maxTotal))

	// Grade distribution chart
	b.WriteString(m.renderGradeDistribution(test) + "\n")

	// Per-question statistics
	b.WriteString(subtitleStyle.Render("Per-Question Statistics") + "\n\n")
	for _, q := range test.Questions {
		sum := 0.0
		min := q.MaxPoints
		max := 0.0
		count := 0

		for _, score := range test.StudentScores {
			points := score.QuestionScores[q.ID]
			sum += points
			if points < min {
				min = points
			}
			if points > max {
				max = points
			}
			count++
		}

		avg := 0.0
		if count > 0 {
			avg = sum / float64(count)
		} else {
			min = 0
		}

		percentage := 0.0
		if q.MaxPoints > 0 {
			percentage = (avg / q.MaxPoints) * 100
		}

		b.WriteString(fmt.Sprintf("  %s (%.0f pts):\n", q.Title, q.MaxPoints))
		b.WriteString(fmt.Sprintf("    Avg: %.1f (%.0f%%)  Min: %.1f  Max: %.1f\n\n",
			avg, percentage, min, max))
	}

	// Help text
	b.WriteString(helpStyle.Render("esc: back to test review"))

	return baseStyle.Render(b.String())
}
