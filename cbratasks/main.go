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
	var listFlag string

	addCmd := &cobra.Command{
		Use:   "add [task title]",
		Short: "Add a new task",
		Long: `Add a new task with optional flags.

Examples:
  cbratasks add "Buy groceries"
  cbratasks add "Meeting with John" --due tomorrow
  cbratasks add "Fix bug" --due +3d --list work
  cbratasks add "Weekend project" --due nextweek --list personal
  cbratasks add "Call mom" --note "Ask about birthday plans"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args, dueFlag, listFlag)
		},
	}

	addCmd.Flags().StringVarP(&dueFlag, "due", "d", "", "Due date (+1d, +1w, tomorrow, nextweek, DD-MM-YYYY)")
	addCmd.Flags().StringVarP(&listFlag, "list", "l", "", "Task list (inbox, work, personal, etc.)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE:  runList,
	}

	// List management commands
	listMgmtCmd := &cobra.Command{
		Use:   "lists",
		Short: "Manage task lists",
	}

	listLsCmd := &cobra.Command{
		Use:   "ls",
		Short: "Show all configured lists",
		RunE:  runListLs,
	}

	listAddCmd := &cobra.Command{
		Use:   "add <name> <color>",
		Short: "Add a new list",
		Long: `Add a new task list with a color.

Color should be in hex format (e.g., #FF6B6B).

Examples:
  cbratasks lists add work #FF6B6B
  cbratasks lists add home #4ECDC4`,
		Args: cobra.ExactArgs(2),
		RunE: runListAdd,
	}

	listRemoveCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a list",
		Args:  cobra.ExactArgs(1),
		RunE:  runListRemove,
	}

	listSetDefaultCmd := &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default list for new tasks",
		Args:  cobra.ExactArgs(1),
		RunE:  runListSetDefault,
	}

	listMgmtCmd.AddCommand(listLsCmd)
	listMgmtCmd.AddCommand(listAddCmd)
	listMgmtCmd.AddCommand(listRemoveCmd)
	listMgmtCmd.AddCommand(listSetDefaultCmd)

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
	rootCmd.AddCommand(listMgmtCmd)
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

func runAdd(args []string, dueFlag string, listFlag string) error {
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

	// Parse due date
	if dueFlag != "" {
		due, err := task.ParseDueDate(dueFlag)
		if err != nil {
			return fmt.Errorf("invalid due date: %w", err)
		}
		newTask.SetDueDate(*due)
	}

	// Save the task (with sync if enabled)
	if err := store.AddTaskWithSync(newTask); err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	// Print confirmation
	fmt.Printf("✓ Added: %s\n", newTask.Title)
	fmt.Printf("  ID: %s\n", newTask.ID)
	fmt.Printf("  List: %s\n", newTask.List)

	if newTask.DueDate != nil {
		fmt.Printf("  Due: %s\n", newTask.DueString())
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

		if t.DueDate != nil {
			line += fmt.Sprintf(" [%s]", t.DueString())
		}

		if t.List != "" {
			line += fmt.Sprintf(" (%s)", t.List)
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

		if t.List != "" {
			line += fmt.Sprintf(" [%s]", t.List)
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

	fmt.Printf("✓ Sync complete! (%d total tasks)\n", len(tasks))

	return nil
}

func runListLs(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	lists := cfg.GetAllLists()
	if len(lists) == 0 {
		fmt.Println("No lists configured.")
		return nil
	}

	fmt.Println("📋 Task Lists:")
	fmt.Println()

	for _, name := range lists {
		color := cfg.GetListColor(name)
		defaultMarker := ""
		if name == cfg.DefaultList {
			defaultMarker = " (default)"
		}
		fmt.Printf("  • %s - %s%s\n", name, color, defaultMarker)
	}

	fmt.Println()
	fmt.Printf("Total: %d lists\n", len(lists))

	return nil
}

func runListAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))
	color := strings.TrimSpace(args[1])

	if name == "" {
		return fmt.Errorf("list name cannot be empty")
	}

	// Validate color format (basic check for # prefix)
	if !strings.HasPrefix(color, "#") {
		return fmt.Errorf("color must be in hex format (e.g., #FF6B6B)")
	}

	// Check if list already exists
	if cfg.ListExists(name) {
		return fmt.Errorf("list '%s' already exists", name)
	}

	cfg.AddList(name, color)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Added list '%s' with color %s\n", name, color)

	return nil
}

func runListRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))

	if !cfg.ListExists(name) {
		return fmt.Errorf("list '%s' does not exist", name)
	}

	if name == cfg.DefaultList {
		return fmt.Errorf("cannot remove default list, set a different default list first")
	}

	cfg.RemoveList(name)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Removed list '%s'\n", name)
	fmt.Println("Note: Tasks in this list will not be deleted, but the list won't show a color.")

	return nil
}

func runListSetDefault(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := strings.ToLower(strings.TrimSpace(args[0]))

	if !cfg.ListExists(name) {
		return fmt.Errorf("list '%s' does not exist, add it first with: cbratasks lists add %s <color>", name, name)
	}

	cfg.DefaultList = name

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Set default list to '%s'\n", name)

	return nil
}
