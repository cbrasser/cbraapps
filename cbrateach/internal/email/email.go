package email

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cbrateach/internal/config"
	"cbrateach/internal/models"
)

// FeedbackEmail represents an email to be sent to a student
type FeedbackEmail struct {
	StudentName string
	StudentEmail string
	Subject string
	Body string
	Attachments []string
}

// ProcessTemplate replaces placeholders in template with actual values
func ProcessTemplate(template string, studentName, testName, courseName string, grade float64, customMessage string) string {
	processed := template
	processed = strings.ReplaceAll(processed, "{{StudentName}}", studentName)
	processed = strings.ReplaceAll(processed, "{{TestName}}", testName)
	processed = strings.ReplaceAll(processed, "{{CourseName}}", courseName)
	processed = strings.ReplaceAll(processed, "{{Grade}}", fmt.Sprintf("%.2f", grade))
	processed = strings.ReplaceAll(processed, "{{CustomMessage}}", customMessage)
	return processed
}

// FindFeedbackFiles finds all files in the directory that match the student's email
// Uses exact matching based on email prefix
func FindFeedbackFiles(directory, studentEmail string) ([]string, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Generate expected filename from email: get part before @, replace dots with dashes
	emailPrefix := strings.Split(studentEmail, "@")[0]
	emailPrefix = strings.ReplaceAll(emailPrefix, ".", "-")
	expectedFilename := emailPrefix + "feedback.txt"

	var exactMatches []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Check if file is not empty (pop requires non-empty attachments)
		fullPath := filepath.Join(directory, file.Name())
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			continue // Skip files we can't stat
		}
		if fileInfo.Size() == 0 {
			continue // Skip empty files
		}

		// Exact match only
		if strings.ToLower(file.Name()) == strings.ToLower(expectedFilename) {
			exactMatches = append(exactMatches, fullPath)
		}
	}

	return exactMatches, nil
}

// sanitizeFilenameForEmail sanitizes student name for file matching
// Must match the sanitization used when creating feedback files
func sanitizeFilenameForEmail(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	return s
}

// normalizeString removes spaces, dashes, underscores and converts to lowercase
func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// PrepareFeedbackEmails prepares emails for all students in a test
func PrepareFeedbackEmails(
	cfg config.Config,
	test models.Test,
	course models.Course,
	feedbackDir string,
	customMessage string,
) ([]FeedbackEmail, error) {
	// Load template
	templatePath := filepath.Join(cfg.MailTemplatesDir(), "feedback_template.txt")
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}
	template := string(templateData)

	var emails []FeedbackEmail

	// Process each student score
	for _, score := range test.StudentScores {
		// Find student email from course using fuzzy matching
		studentEmail := ""
		normalizedScoreName := normalizeString(score.StudentName)

		for _, student := range course.Students {
			normalizedStudentName := normalizeString(student.Name)
			// Check if the course student name appears in the score name
			// This handles cases like "Claudio Brasser" matching "Claudio Brasser 8.5"
			if strings.Contains(normalizedScoreName, normalizedStudentName) ||
			   strings.Contains(normalizedStudentName, normalizedScoreName) {
				studentEmail = student.Email
				break
			}
		}

		if studentEmail == "" {
			// Skip students without email
			continue
		}

		// Process template
		body := ProcessTemplate(template, score.StudentName, test.Title, course.Name, score.Grade, customMessage)

		// Find attachments using email
		attachments, err := FindFeedbackFiles(feedbackDir, studentEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to find feedback files for %s: %w", score.StudentName, err)
		}

		emails = append(emails, FeedbackEmail{
			StudentName: score.StudentName,
			StudentEmail: studentEmail,
			Subject: fmt.Sprintf("[%s] Test Feedback: %s", course.Name, test.Title),
			Body: body,
			Attachments: attachments,
		})
	}

	return emails, nil
}

// EmailSummary provides a summary of prepared emails for confirmation
func EmailSummary(emails []FeedbackEmail) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Ready to send %d emails:\n\n", len(emails)))

	for i, email := range emails {
		b.WriteString(fmt.Sprintf("%d. %s <%s>\n", i+1, email.StudentName, email.StudentEmail))
		b.WriteString(fmt.Sprintf("   Subject: %s\n", email.Subject))
		b.WriteString(fmt.Sprintf("   Attachments: %d file(s)\n", len(email.Attachments)))
		if len(email.Attachments) > 0 {
			for _, att := range email.Attachments {
				b.WriteString(fmt.Sprintf("     - %s\n", filepath.Base(att)))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// EmailPreview formats a single email for preview
func EmailPreview(email FeedbackEmail, bccEmail string, isFirst bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("To: %s <%s>\n", email.StudentName, email.StudentEmail))
	if isFirst && bccEmail != "" {
		b.WriteString(fmt.Sprintf("BCC: %s (first email only)\n", bccEmail))
	}
	b.WriteString(fmt.Sprintf("Subject: %s\n\n", email.Subject))
	b.WriteString("---\n\n")
	b.WriteString(email.Body)
	b.WriteString("\n\n---\n\n")
	b.WriteString(fmt.Sprintf("Attachments (%d):\n", len(email.Attachments)))
	if len(email.Attachments) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, att := range email.Attachments {
			b.WriteString(fmt.Sprintf("  - %s\n", filepath.Base(att)))
		}
	}

	return b.String()
}
