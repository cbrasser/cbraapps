package main

import (
	"fmt"
	"os"
	"strings"

	"cbratasks/internal/config"
	"cbratasks/internal/storage"
	"cbratasks/internal/task"
	"cbratasks/internal/tui"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cbratasks",
		Short: "A simple task management app",
		Long:  "cbratasks is a minimal task manager with local storage and optional CalDAV sync.",
		RunE:  runTUI,
	}

	// Add command with flags
	var dueFlag string
	var tagsFlag []string
	var listFlag string
	var noteFlag string

	addCmd := &cobra.Command{
		Use:   "add [task title]",
		Short: "Add a new task",
		Long: `Add a new task with optional flags.

Examples:
  cbratasks add "Buy groceries"
  cbratasks add "Meeting with John" --due tomorrow
  cbratasks add "Fix bug" --due +3d --tag work --tag urgent
  cbratasks add "Weekend project" --due nextweek --tag home
  cbratasks add "Call mom" --note "Ask about birthday plans"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args, dueFlag, tagsFlag, listFlag, noteFlag)
		},
	}

	addCmd.Flags().StringVarP(&dueFlag, "due", "d", "", "Due date (+1d, +1w, tomorrow, nextweek, DD-MM-YYYY)")
	addCmd.Flags().StringSliceVarP(&tagsFlag, "tag", "T", nil, "Tags (can be specified multiple times)")
	addCmd.Flags().StringVarP(&listFlag, "list", "l", "", "Task list (local or radicale)")
	addCmd.Flags().StringVarP(&noteFlag, "note", "n", "", "Attach a note to the task")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE:  runList,
	}

	todayCmd := &cobra.Command{
		Use:   "today",
		Short: "List tasks due today",
		Long: `List all incomplete tasks that are due today.

Useful for scripts, integrations, or quick overview of what needs to be done.

Output format (one task per line):
  - Task title [tags] (ID)`,
		RunE: runToday,
	}

	archiveCmd := &cobra.Command{
		Use:   "archive",
		Short: "Show archived tasks",
		RunE:  runArchive,
	}

	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync tasks with CalDAV server (Radicale)",
		Long: `Synchronize tasks with a CalDAV server like Radicale.

The server URL, username, and password must be configured in the config file.
A 'cbratasks' collection will be created automatically if it doesn't exist.`,
		RunE: runSync,
	}

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(todayCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(syncCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	return tui.Run(cfg, store)
}

func runAdd(args []string, dueFlag string, tagsFlag []string, listFlag string, noteFlag string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Determine which list to use
	listName := cfg.DefaultList
	if listFlag != "" {
		listName = listFlag
	}

	// Create the task
	title := strings.Join(args, " ")
	newTask := task.NewTask(title, listName)

	// Add tags
	for _, tag := range tagsFlag {
		newTask.AddTag(tag)
	}

	// Parse due date
	if dueFlag != "" {
		due, err := task.ParseDueDate(dueFlag)
		if err != nil {
			return fmt.Errorf("invalid due date: %w", err)
		}
		newTask.SetDueDate(*due)
	}

	// Add note
	if noteFlag != "" {
		newTask.SetNote(noteFlag)
	}

	// Save the task (with sync if radicale)
	if err := store.AddTaskWithSync(newTask); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	// Print confirmation
	fmt.Printf("✓ Added: %s\n", newTask.Title)
	fmt.Printf("  ID: %s\n", newTask.ID)

	if newTask.DueDate != nil {
		fmt.Printf("  Due: %s\n", newTask.DueString())
	}

	if len(newTask.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(newTask.Tags, ", "))
	}

	if newTask.HasNote() {
		fmt.Printf("  Note: %s\n", newTask.Note)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	// Ensure config exists
	if _, err := config.Load(); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	tasks := store.GetTasks()

	if len(tasks) == 0 {
		fmt.Println("No tasks. Add one with: cbratasks add \"task name\"")
		return nil
	}

	fmt.Println("📋 Tasks:")
	fmt.Println()

	for _, t := range tasks {
		checkbox := "[ ]"
		if t.Completed {
			checkbox = "[x]"
		}

		line := fmt.Sprintf("  %s %s", checkbox, t.Title)

		if t.HasNote() {
			line += " 📝"
		}

		if t.DueDate != nil {
			line += fmt.Sprintf(" [%s]", t.DueString())
		}

		if len(t.Tags) > 0 {
			line += fmt.Sprintf(" (%s)", strings.Join(t.Tags, ", "))
		}

		if t.IsOverdue() {
			line += " ⚠ OVERDUE"
		}

		fmt.Println(line)
	}

	fmt.Println()
	fmt.Printf("Total: %d tasks\n", len(tasks))

	// Show config location on first run
	if !config.Exists() {
		fmt.Printf("\nConfig created at: %s\n", config.ConfigPath())
	}

	return nil
}

func runToday(cmd *cobra.Command, args []string) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	tasks := store.GetTasksDueToday()

	if len(tasks) == 0 {
		// Output nothing for scripts - empty means no tasks due today
		return nil
	}

	// Simple output format for scripts/integrations
	for _, t := range tasks {
		line := fmt.Sprintf("- %s", t.Title)

		if len(t.Tags) > 0 {
			line += fmt.Sprintf(" [%s]", strings.Join(t.Tags, ", "))
		}

		line += fmt.Sprintf(" (%s)", t.ID)

		fmt.Println(line)
	}

	return nil
}

func runArchive(cmd *cobra.Command, args []string) error {
	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	archived := store.GetArchivedTasks()

	if len(archived) == 0 {
		fmt.Println("No archived tasks.")
		return nil
	}

	fmt.Println("📦 Archived Tasks:")
	fmt.Println()

	for _, t := range archived {
		line := fmt.Sprintf("  [x] %s", t.Title)

		if t.CompletedAt != nil {
			line += fmt.Sprintf(" (completed %s)", t.CompletedAt.Format("02 Jan 2006"))
		}

		fmt.Println(line)
	}

	fmt.Println()
	fmt.Printf("Total: %d archived tasks\n", len(archived))

	return nil
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.Sync.Enabled {
		fmt.Println("Sync is not enabled. Enable it in the config file:")
		fmt.Printf("  %s\n", config.ConfigPath())
		fmt.Println()
		fmt.Println("Set [sync] enabled = true and configure URL, username, password.")
		return nil
	}

	if cfg.Sync.URL == "" {
		return fmt.Errorf("sync URL not configured")
	}

	store, err := storage.New()
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	fmt.Println("🔄 Syncing with CalDAV server...")
	fmt.Printf("   Server: %s\n", cfg.Sync.URL)

	if err := store.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Show synced tasks count
	tasks := store.GetTasks()
	radicaleCount := 0
	for _, t := range tasks {
		if t.ListName == "radicale" {
			radicaleCount++
		}
	}

	fmt.Printf("✓ Sync complete! (%d tasks from server)\n", radicaleCount)

	return nil
}
