package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"mytuiapp/internal/notify"
)

func main() {
	//TODO: Flag "--tomorrow" -> Show tomorrow at a glance
	nextFlag := flag.Bool("next", false, "Show next upcoming event and quit")
	dayFlag := flag.Bool("day", false, "Show daily view and quit")
	weekFlag := flag.Bool("week", false, "Show weekly view and quit")
	monthFlag := flag.Bool("month", false, "Show monthly view and quit")
	listFlag := flag.String("list", "", "List events for a specific day (format: YYYY-MM-DD, 'today', 'tomorrow', or empty for today)")
	listTodayFlag := flag.Bool("today", false, "List today's events (shortcut for --list today)")
	jsonFlag := flag.Bool("json", false, "Output in JSON format (use with --list or --today)")
	daemonFlag := flag.Bool("daemon", false, "Run notification daemon in the background")
	flag.Parse()

	config, _ := loadConfig()
	var radicaleConfig *RadicaleConfig
	if config != nil && config.Radicale != nil {
		radicaleConfig = config.Radicale
	}

	// Handle --daemon flag
	if *daemonFlag {
		if config == nil || config.Notifications == nil {
			fmt.Println("Error: No notification configuration found")
			return
		}
		if !config.Notifications.Enabled {
			fmt.Println("Error: Notifications are disabled in config")
			return
		}
		runDaemon(config.Notifications, radicaleConfig)
		return
	}

	// Handle --list and --today flags
	if *listTodayFlag || flag.Lookup("list").Value.String() != "" || *listFlag != "" {
		events, _, _, err := loadAllCalendars(radicaleConfig)
		if err != nil {
			fmt.Printf("Error loading calendars: %v\n", err)
			return
		}

		// Determine target date
		targetDate := time.Now()
		dateStr := *listFlag
		if *listTodayFlag {
			dateStr = "today"
		}

		if dateStr != "" && dateStr != "today" {
			if dateStr == "tomorrow" {
				targetDate = time.Now().AddDate(0, 0, 1)
			} else {
				parsed, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					fmt.Printf("Invalid date format: %s (use YYYY-MM-DD, 'today', or 'tomorrow')\n", dateStr)
					return
				}
				targetDate = parsed
			}
		}

		// Filter and output events
		dayEvents := getEventsForDay(events, targetDate)
		if *jsonFlag {
			fmt.Println(formatEventsJSON(dayEvents))
		} else {
			fmt.Print(formatEventsList(dayEvents, targetDate))
		}
		return
	}

	// For one-shot modes, we need to load calendars synchronously
	if *nextFlag || *dayFlag || *weekFlag || *monthFlag {
		events, calendars, calendarURLs, _ := loadAllCalendars(radicaleConfig)

		if *nextFlag {
			nextEvent := getNextEvent(events)
			fmt.Println(renderNextEvent(nextEvent))
			return
		}

		viewMode := DailyView
		if *weekFlag {
			viewMode = WeeklyView
		} else if *monthFlag {
			viewMode = MonthlyView
		}

		m := initialModel(viewMode, true, radicaleConfig)
		m.events = events
		m.calendars = calendars
		m.calendarURLs = calendarURLs
		m.isLoading = false
		// Set default selected calendar
		for name := range m.calendars {
			m.selectedCalendar = name
			break
		}

		fmt.Println(m.View())
		return
	}

	// Interactive mode - load calendars async with spinner
	m := initialModel(DailyView, false, radicaleConfig)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// getEventsForDay returns all events that occur on the specified day
func getEventsForDay(events []Event, day time.Time) []Event {
	var dayEvents []Event
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	for _, event := range events {
		// Event occurs on this day if:
		// - It starts on this day, OR
		// - It spans across this day (started before, ends after dayStart)
		startsOnDay := event.Start.After(dayStart.Add(-time.Second)) && event.Start.Before(dayEnd)
		spansDay := event.Start.Before(dayStart) && event.End.After(dayStart)

		if startsOnDay || spansDay {
			dayEvents = append(dayEvents, event)
		}
	}

	// Sort by start time
	sort.Slice(dayEvents, func(i, j int) bool {
		return dayEvents[i].Start.Before(dayEvents[j].Start)
	})

	return dayEvents
}

// formatEventsList formats events as plain text for shell scripts
func formatEventsList(events []Event, day time.Time) string {
	if len(events) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, event := range events {
		startTime := event.Start.Format("15:04")
		endTime := event.End.Format("15:04")
		duration := formatDuration(event.End.Sub(event.Start))

		sb.WriteString(fmt.Sprintf("%s-%s (%s) %s\n", startTime, endTime, duration, event.Summary))
	}

	return sb.String()
}

// formatEventsJSON formats events as JSON for programmatic use
func formatEventsJSON(events []Event) string {
	if len(events) == 0 {
		return "[]"
	}

	var sb strings.Builder
	sb.WriteString("[\n")
	for i, event := range events {
		duration := formatDuration(event.End.Sub(event.Start))
		// Escape JSON strings
		title := strings.ReplaceAll(event.Summary, `"`, `\"`)
		title = strings.ReplaceAll(title, "\n", "\\n")

		sb.WriteString(fmt.Sprintf(`  {"title":"%s","start":"%s","end":"%s","duration":"%s","calendar":"%s"}`,
			title,
			event.Start.Format("15:04"),
			event.End.Format("15:04"),
			duration,
			event.CalendarName,
		))
		if i < len(events)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]")

	return sb.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// runDaemon starts the notification daemon
func runDaemon(notifConfig *NotificationConfig, radicaleConfig *RadicaleConfig) {
	// Create event loader function that wraps loadAllCalendars
	loader := func() ([]notify.Event, error) {
		events, _, _, err := loadAllCalendars(radicaleConfig)
		if err != nil {
			return nil, err
		}

		// Convert main.Event to notify.Event
		notifyEvents := make([]notify.Event, len(events))
		for i, e := range events {
			notifyEvents[i] = notify.Event{
				Summary:      e.Summary,
				Start:        e.Start,
				End:          e.End,
				Description:  e.Description,
				CalendarName: e.CalendarName,
				UID:          e.UID,
			}
		}
		return notifyEvents, nil
	}

	// Create notify config
	config := &notify.NotificationConfig{
		Enabled:        notifConfig.Enabled,
		CheckInterval:  notifConfig.CheckInterval,
		AdvanceNotice:  notifConfig.AdvanceNotice,
		ReloadInterval: notifConfig.ReloadInterval,
	}

	// Create and run daemon
	daemon := notify.NewDaemon(config, loader)
	if err := daemon.Run(); err != nil {
		log.Fatalf("Daemon error: %v", err)
	}
}
