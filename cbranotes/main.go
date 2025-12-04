package main

import (
	"fmt"
	"os"

	"cbranotes/internal/config"
	"cbranotes/internal/git"
	"cbranotes/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cbranotes",
		Short: "A minimal notes sync tool",
		Long:  "cbranotes syncs your notes through git with a minimal TUI.",
	}

	var syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Sync notes with remote repository",
	}

	var upCmd = &cobra.Command{
		Use:   "up",
		Short: "Commit all changes and push to remote",
		RunE:  runSyncUp,
	}

	var downCmd = &cobra.Command{
		Use:   "down",
		Short: "Pull latest changes from remote",
		RunE:  runSyncDown,
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show sync status (unpushed/unpulled changes)",
		RunE:  runSyncStatus,
	}

	var editCmd = &cobra.Command{
		Use:   "edit",
		Short: "Open the note editor",
		RunE:  runEdit,
	}

	syncCmd.AddCommand(upCmd)
	syncCmd.AddCommand(downCmd)
	syncCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(editCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func ensureConfig() (*config.Config, error) {
	if !config.Exists() {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		model := tui.NewSetupModel(cwd)
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return nil, err
		}

		setupModel := finalModel.(tui.SetupModel)
		if !setupModel.Done() {
			return nil, fmt.Errorf("setup cancelled")
		}

		cfg := &config.Config{
			RepoURL:   setupModel.RepoURL,
			NotesPath: setupModel.NotesPath,
		}

		if err := config.Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		// Clone the repository
		if !git.IsRepo(cfg.NotesPath) {
			spinnerModel := tui.NewSpinnerModel("Cloning repository", func() error {
				return git.Clone(cfg.RepoURL, cfg.NotesPath)
			})
			p := tea.NewProgram(spinnerModel)
			finalModel, err := p.Run()
			if err != nil {
				return nil, err
			}
			if spinnerErr := finalModel.(tui.SpinnerModel).Err(); spinnerErr != nil {
				return nil, spinnerErr
			}
		}

		return cfg, nil
	}

	return config.Load()
}

func runSyncUp(cmd *cobra.Command, args []string) error {
	cfg, err := ensureConfig()
	if err != nil {
		return err
	}

	if !git.IsRepo(cfg.NotesPath) {
		return fmt.Errorf("notes directory is not a git repository: %s", cfg.NotesPath)
	}

	// Check for changes
	hasChanges, err := git.HasChanges(cfg.NotesPath)
	if err != nil {
		return err
	}

	if !hasChanges {
		fmt.Println("No changes to sync")
		return nil
	}

	// Commit and push
	spinnerModel := tui.NewSpinnerModel("Syncing up", func() error {
		if err := git.CommitAll(cfg.NotesPath); err != nil {
			return err
		}
		return git.Push(cfg.NotesPath)
	})

	p := tea.NewProgram(spinnerModel)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if spinnerErr := finalModel.(tui.SpinnerModel).Err(); spinnerErr != nil {
		return spinnerErr
	}

	return nil
}

func runSyncDown(cmd *cobra.Command, args []string) error {
	cfg, err := ensureConfig()
	if err != nil {
		return err
	}

	if !git.IsRepo(cfg.NotesPath) {
		return fmt.Errorf("notes directory is not a git repository: %s", cfg.NotesPath)
	}

	spinnerModel := tui.NewSpinnerModel("Syncing down", func() error {
		return git.Pull(cfg.NotesPath)
	})

	p := tea.NewProgram(spinnerModel)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if spinnerErr := finalModel.(tui.SpinnerModel).Err(); spinnerErr != nil {
		return spinnerErr
	}

	return nil
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	cfg, err := ensureConfig()
	if err != nil {
		return err
	}

	if !git.IsRepo(cfg.NotesPath) {
		return fmt.Errorf("notes directory is not a git repository: %s", cfg.NotesPath)
	}

	status, err := git.GetSyncStatus(cfg.NotesPath)
	if err != nil {
		return err
	}

	// Display status
	hasOutput := false

	if len(status.LocalChanges) > 0 {
		fmt.Println("📝 Local changes (uncommitted):")
		for _, change := range status.LocalChanges {
			fmt.Printf("   %s\n", change)
		}
		hasOutput = true
	}

	if len(status.UnpushedCommits) > 0 {
		if hasOutput {
			fmt.Println()
		}
		fmt.Printf("⬆ Unpushed commits (%d):\n", len(status.UnpushedCommits))
		for _, commit := range status.UnpushedCommits {
			fmt.Printf("   %s\n", commit)
		}
		hasOutput = true
	}

	if len(status.UnpulledCommits) > 0 {
		if hasOutput {
			fmt.Println()
		}
		fmt.Printf("⬇ Unpulled commits (%d):\n", len(status.UnpulledCommits))
		for _, commit := range status.UnpulledCommits {
			fmt.Printf("   %s\n", commit)
		}
		hasOutput = true
	}

	if !hasOutput {
		fmt.Println("✓ Everything is in sync!")
	}

	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	cfg, err := ensureConfig()
	if err != nil {
		return err
	}

	// Check if notes path exists
	if _, err := os.Stat(cfg.NotesPath); os.IsNotExist(err) {
		return fmt.Errorf("notes directory does not exist: %s", cfg.NotesPath)
	}

	editorModel := tui.NewEditorModel(cfg)
	p := tea.NewProgram(editorModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

