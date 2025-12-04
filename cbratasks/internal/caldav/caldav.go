package caldav

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cbratasks/internal/task"

	"github.com/google/uuid"
)

const collectionName = "cbratasks"

type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

func NewClient(baseURL, username, password string) *Client {
	// Ensure baseURL doesn't end with slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) collectionURL() string {
	return fmt.Sprintf("%s/%s/%s/", c.baseURL, c.username, collectionName)
}

func (c *Client) doRequest(method, url string, body []byte, contentType string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return c.client.Do(req)
}

// EnsureCollection creates the cbratasks collection if it doesn't exist
func (c *Client) EnsureCollection() error {
	// Check if collection exists with PROPFIND
	resp, err := c.doRequest("PROPFIND", c.collectionURL(), nil, "")
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 207 {
		// Collection exists
		return nil
	}

	if resp.StatusCode == 404 {
		// Create the collection
		return c.createCollection()
	}

	return fmt.Errorf("unexpected status checking collection: %d", resp.StatusCode)
}

func (c *Client) createCollection() error {
	// MKCALENDAR request body for a VTODO collection
	body := `<?xml version="1.0" encoding="UTF-8"?>
<mkcalendar xmlns="urn:ietf:params:xml:ns:caldav">
  <set xmlns="DAV:">
    <prop>
      <displayname>cbratasks</displayname>
      <calendar-description xmlns="urn:ietf:params:xml:ns:caldav">Task list managed by cbratasks</calendar-description>
      <supported-calendar-component-set xmlns="urn:ietf:params:xml:ns:caldav">
        <comp name="VTODO"/>
      </supported-calendar-component-set>
    </prop>
  </set>
</mkcalendar>`

	resp, err := c.doRequest("MKCALENDAR", c.collectionURL(), []byte(body), "application/xml")
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetAllTasks fetches all tasks from the CalDAV server
func (c *Client) GetAllTasks() ([]*task.Task, error) {
	// REPORT request to get all VTODOs
	body := `<?xml version="1.0" encoding="UTF-8"?>
<calendar-query xmlns="urn:ietf:params:xml:ns:caldav" xmlns:d="DAV:">
  <d:prop>
    <d:getetag/>
    <calendar-data/>
  </d:prop>
  <filter>
    <comp-filter name="VCALENDAR">
      <comp-filter name="VTODO"/>
    </comp-filter>
  </filter>
</calendar-query>`

	req, err := http.NewRequest("REPORT", c.collectionURL(), bytes.NewReader([]byte(body)))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Depth", "1")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 207 {
		return nil, fmt.Errorf("failed to fetch tasks: status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tasks, err := parseMultistatusResponse(string(respBody))
	if err != nil {
		return nil, err
	}

	// If no tasks found but we got data, try extracting VTODOs directly
	if len(tasks) == 0 && strings.Contains(string(respBody), "BEGIN:VTODO") {
		tasks = extractVTODOsDirectly(string(respBody))
	}

	return tasks, nil
}

// CreateTask creates a new task on the CalDAV server
func (c *Client) CreateTask(t *task.Task) error {
	ical := taskToVTODO(t)
	url := fmt.Sprintf("%s%s.ics", c.collectionURL(), t.ID)

	resp, err := c.doRequest("PUT", url, []byte(ical), "text/calendar; charset=utf-8")
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 204 && resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create task: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// UpdateTask updates an existing task on the CalDAV server
func (c *Client) UpdateTask(t *task.Task) error {
	return c.CreateTask(t) // PUT is idempotent
}

// DeleteTask deletes a task from the CalDAV server
func (c *Client) DeleteTask(id string) error {
	url := fmt.Sprintf("%s%s.ics", c.collectionURL(), id)

	resp, err := c.doRequest("DELETE", url, nil, "")
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("failed to delete task: status %d", resp.StatusCode)
	}

	return nil
}

// taskToVTODO converts a Task to iCalendar VTODO format
func taskToVTODO(t *task.Task) string {
	var b strings.Builder

	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//cbratasks//EN\r\n")
	b.WriteString("BEGIN:VTODO\r\n")

	// UID
	b.WriteString(fmt.Sprintf("UID:%s\r\n", t.ID))

	// Timestamps
	b.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICalTime(time.Now())))
	b.WriteString(fmt.Sprintf("CREATED:%s\r\n", formatICalTime(t.CreatedAt)))
	b.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", formatICalTime(t.UpdatedAt)))

	// Summary (title)
	b.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(t.Title)))

	// Description (note) - this is how notes sync with CalDAV
	if t.Note != "" {
		b.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(t.Note)))
	}

	// Due date - use full datetime in UTC for mobile app compatibility
	if t.DueDate != nil {
		// Use UTC datetime format - most compatible with mobile apps
		// Set to end of day (23:59:59) in UTC
		dueUTC := t.DueDate.UTC()
		b.WriteString(fmt.Sprintf("DUE:%s\r\n", dueUTC.Format("20060102T150405Z")))
	}

	// Status
	if t.Completed {
		b.WriteString("STATUS:COMPLETED\r\n")
		if t.CompletedAt != nil {
			b.WriteString(fmt.Sprintf("COMPLETED:%s\r\n", formatICalTime(*t.CompletedAt)))
		}
		b.WriteString("PERCENT-COMPLETE:100\r\n")
	} else {
		b.WriteString("STATUS:NEEDS-ACTION\r\n")
		b.WriteString("PERCENT-COMPLETE:0\r\n")
	}

	// Categories (tags)
	if len(t.Tags) > 0 {
		b.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", strings.Join(t.Tags, ",")))
	}

	b.WriteString("END:VTODO\r\n")
	b.WriteString("END:VCALENDAR\r\n")

	return b.String()
}

// parseMultistatusResponse parses a CalDAV multistatus response
func parseMultistatusResponse(body string) ([]*task.Task, error) {
	var tasks []*task.Task

	// Try multiple namespace patterns for calendar-data
	patterns := []string{
		`(?s)<cal:calendar-data[^>]*>(.*?)</cal:calendar-data>`,
		`(?s)<C:calendar-data[^>]*>(.*?)</C:calendar-data>`,
		`(?s)<calendar-data[^>]*>(.*?)</calendar-data>`,
		`(?s)<ns\d:calendar-data[^>]*>(.*?)</ns\d:calendar-data>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(body, -1)

		for _, match := range matches {
			if len(match) > 1 {
				icalData := unescapeXML(match[1])
				t, err := vtodoToTask(icalData)
				if err != nil {
					continue // Skip invalid entries
				}
				tasks = append(tasks, t)
			}
		}
	}

	return tasks, nil
}

// extractVTODOsDirectly extracts VTODOs directly from raw response as fallback
func extractVTODOsDirectly(body string) []*task.Task {
	var tasks []*task.Task

	// Find all BEGIN:VCALENDAR...END:VCALENDAR blocks or BEGIN:VTODO...END:VTODO
	re := regexp.MustCompile(`(?s)(BEGIN:VCALENDAR.*?END:VCALENDAR|BEGIN:VTODO.*?END:VTODO)`)
	matches := re.FindAllString(body, -1)

	for _, match := range matches {
		// Unescape XML entities if present
		match = unescapeXML(match)
		t, err := vtodoToTask(match)
		if err != nil {
			continue
		}
		tasks = append(tasks, t)
	}

	return tasks
}

// vtodoToTask converts iCalendar VTODO to a Task
func vtodoToTask(ical string) (*task.Task, error) {
	t := &task.Task{
		ID:        uuid.New().String(),
		ListName:  "radicale",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	lines := strings.Split(ical, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, "\r")

		if strings.HasPrefix(line, "UID:") {
			t.ID = strings.TrimPrefix(line, "UID:")
		} else if strings.HasPrefix(line, "SUMMARY:") {
			t.Title = unescapeICalText(strings.TrimPrefix(line, "SUMMARY:"))
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			t.Note = unescapeICalText(strings.TrimPrefix(line, "DESCRIPTION:"))
		} else if strings.HasPrefix(line, "DUE") {
			due := parseDueLine(line)
			if due != nil {
				t.DueDate = due
			}
		} else if strings.HasPrefix(line, "STATUS:") {
			status := strings.TrimPrefix(line, "STATUS:")
			t.Completed = (status == "COMPLETED")
		} else if strings.HasPrefix(line, "COMPLETED:") {
			completed := parseICalTime(strings.TrimPrefix(line, "COMPLETED:"))
			if completed != nil {
				t.CompletedAt = completed
			}
		} else if strings.HasPrefix(line, "CATEGORIES:") {
			cats := strings.TrimPrefix(line, "CATEGORIES:")
			t.Tags = strings.Split(cats, ",")
		} else if strings.HasPrefix(line, "CREATED:") {
			created := parseICalTime(strings.TrimPrefix(line, "CREATED:"))
			if created != nil {
				t.CreatedAt = *created
			}
		} else if strings.HasPrefix(line, "LAST-MODIFIED:") {
			modified := parseICalTime(strings.TrimPrefix(line, "LAST-MODIFIED:"))
			if modified != nil {
				t.UpdatedAt = *modified
			}
		}
	}

	if t.Title == "" {
		return nil, fmt.Errorf("task has no title")
	}

	return t, nil
}

func parseDueLine(line string) *time.Time {
	// Handle DUE;VALUE=DATE:20240115 or DUE:20240115T120000Z
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	dateStr := strings.TrimSpace(parts[1])

	// Try DATE format first (YYYYMMDD)
	if len(dateStr) == 8 {
		if t, err := time.Parse("20060102", dateStr); err == nil {
			// Set to end of day
			t = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.Local)
			return &t
		}
	}

	// Try datetime format
	return parseICalTime(dateStr)
}

func formatICalTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

func parseICalTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	// Try various formats
	formats := []string{
		"20060102T150405Z",
		"20060102T150405",
		"20060102",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return &t
		}
	}
	return nil
}

func escapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func unescapeICalText(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func unescapeXML(s string) string {
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}
