package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cbratasks/internal/caldav"
	"cbratasks/internal/config"
	"cbratasks/internal/task"

	"cbratasks/internal/github"
)

type Storage struct {
	tasks    []*task.Task
<<<<<<< HEAD
=======
	issues   []*github.Issue
>>>>>>> 898c55541e02bb68570a3f243a3a31bb60619efb
	archived []*task.Task
	dataDir  string
	mu       sync.RWMutex
	caldav   *caldav.Client
	cfg      *config.Config
}

func New() (*Storage, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	return NewWithConfig(cfg)
}

func NewWithConfig(cfg *config.Config) (*Storage, error) {
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	s := &Storage{
		dataDir: dataDir,
		cfg:     cfg,
	}

	// Migrate config from tags to lists
	cfg.MigrateTagsToLists()
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save migrated config: %w", err)
	}

	// Initialize CalDAV client if sync is enabled
	if cfg.Sync.Enabled && cfg.Sync.URL != "" {
		client, err := caldav.NewClient(cfg.Sync.URL, cfg.Sync.Username, cfg.Sync.Password, cfg.Sync.Backend)
		if err != nil {
			return nil, fmt.Errorf("failed to create caldav client: %w", err)
		}
		s.caldav = client
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	// Migrate tasks from old format
	s.migrateTasks()

	// Auto-archive old completed tasks
	s.archiveOldTasks()

	return s, nil
}

func (s *Storage) tasksFile() string {
	return filepath.Join(s.dataDir, "tasks.json")
}

func (s *Storage) archiveFile() string {
	return filepath.Join(s.dataDir, "archive.json")
}

func (s *Storage) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load active tasks
	if data, err := os.ReadFile(s.tasksFile()); err == nil {
		if err := json.Unmarshal(data, &s.tasks); err != nil {
			return err
		}
	}

	// Load archived tasks
	if data, err := os.ReadFile(s.archiveFile()); err == nil {
		if err := json.Unmarshal(data, &s.archived); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) save() error {
	// Save active tasks
	data, err := json.MarshalIndent(s.tasks, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.tasksFile(), data, 0o644); err != nil {
		return err
	}

	// Save archived tasks
	archiveData, err := json.MarshalIndent(s.archived, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.archiveFile(), archiveData, 0o644)
}

// archiveOldTasks moves completed tasks older than 24h to archive
func (s *Storage) migrateTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultList := s.cfg.DefaultList
	if defaultList == "" {
		defaultList = "inbox"
	}

	migrated := false
	for _, t := range s.tasks {
		if t.List == "" {
			t.MigrateFromOldFormat(defaultList)
			migrated = true
		}
	}

	for _, t := range s.archived {
		if t.List == "" {
			t.MigrateFromOldFormat(defaultList)
			migrated = true
		}
	}

	if migrated {
		s.save() // Save migrated tasks
	}
}

func (s *Storage) archiveOldTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var active []*task.Task
	for _, t := range s.tasks {
		if t.ShouldArchive() {
			t.Archived = true
			s.archived = append(s.archived, t)
		} else {
			active = append(active, t)
		}
	}
	s.tasks = active
	s.save()
}

// GetTasks returns all active tasks (including recently completed)
func (s *Storage) GetTasks() []*task.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Sort by: incomplete first, then by due date, then by created date
	tasks := make([]*task.Task, len(s.tasks))
	copy(tasks, s.tasks)

	sort.Slice(tasks, func(i, j int) bool {
		// Completed tasks at the bottom
		if tasks[i].Completed != tasks[j].Completed {
			return !tasks[i].Completed
		}

		// Sort by due date (tasks with due dates first)
		if tasks[i].DueDate != nil && tasks[j].DueDate != nil {
			if !tasks[i].DueDate.Equal(*tasks[j].DueDate) {
				return tasks[i].DueDate.Before(*tasks[j].DueDate)
			}
			// Same due date - group by tag
			tagI := ""
			if len(tasks[i].Tags) > 0 {
				tagI = tasks[i].Tags[0]
			}
			tagJ := ""
			if len(tasks[j].Tags) > 0 {
				tagJ = tasks[j].Tags[0]
			}
			if tagI != tagJ {
				return tagI < tagJ
			}
			// Same tag - sort by created date
			return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
		}
		if tasks[i].DueDate != nil {
			return true
		}
		if tasks[j].DueDate != nil {
			return false
		}

		// No due dates - group by tag
		tagI := ""
		if len(tasks[i].Tags) > 0 {
			tagI = tasks[i].Tags[0]
		}
		tagJ := ""
		if len(tasks[j].Tags) > 0 {
			tagJ = tasks[j].Tags[0]
		}
		if tagI != tagJ {
			return tagI < tagJ
		}

		// Sort by created date
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	return tasks
}

// GetTasksDueToday returns all incomplete tasks due today
func (s *Storage) GetTasksDueToday() []*task.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*task.Task
	for _, t := range s.tasks {
		if !t.Completed && t.IsDueToday() {
			results = append(results, t)
		}
	}
	return results
}

// GetTask returns a task by ID
func (s *Storage) GetTask(id string) *task.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// AddTask adds a new task
func (s *Storage) AddTask(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = append(s.tasks, t)
	return s.save()
}

// UpdateTask updates an existing task
func (s *Storage) UpdateTask(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t.UpdatedAt = time.Now()

	for i, existing := range s.tasks {
		if existing.ID == t.ID {
			s.tasks[i] = t
			return s.save()
		}
	}
	return nil
}

// DeleteTask deletes a task by ID
func (s *Storage) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return s.save()
		}
	}
	return nil
}

// ToggleComplete toggles the completion status of a task
func (s *Storage) ToggleComplete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tasks {
		if t.ID == id {
			t.ToggleComplete()
			return s.save()
		}
	}
	return nil
}

// GetArchivedTasks returns all archived tasks
func (s *Storage) GetArchivedTasks() []*task.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	archived := make([]*task.Task, len(s.archived))
	copy(archived, s.archived)
	return archived
}

// Search performs a fuzzy search on task titles
func (s *Storage) Search(query string) []*task.Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		return s.GetTasks()
	}

	var results []*task.Task
	query = strings.ToLower(query)

	for _, t := range s.tasks {
		if fuzzyMatch(strings.ToLower(t.Title), query) {
			results = append(results, t)
		}
	}

	return results
}

// fuzzyMatch performs a simple fuzzy match
func fuzzyMatch(str, pattern string) bool {
	patternIdx := 0
	for i := 0; i < len(str) && patternIdx < len(pattern); i++ {
		if str[i] == pattern[patternIdx] {
			patternIdx++
		}
	}
	return patternIdx == len(pattern)
}

// IsSyncEnabled returns true if CalDAV sync is enabled
func (s *Storage) IsSyncEnabled() bool {
	return s.caldav != nil
}

// Sync synchronizes tasks with the CalDAV server
func (s *Storage) Sync() error {
	if s.caldav == nil {
		return fmt.Errorf("sync not enabled")
	}

	ctx := context.Background()

	// Get all list names from config
	listNames := s.cfg.GetAllLists()
	if len(listNames) == 0 {
		return fmt.Errorf("no lists configured")
	}

	// Pull remote tasks from all lists
	remoteTasks, err := s.caldav.GetAllTasks(ctx, listNames)
	if err != nil {
		return fmt.Errorf("failed to fetch remote tasks: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build map of archived task IDs
	archivedByID := make(map[string]bool)
	for _, t := range s.archived {
		archivedByID[t.ID] = true
	}

	// Build maps for comparison
	localByID := make(map[string]*task.Task)
	for _, t := range s.tasks {
		localByID[t.ID] = t
	}

	remoteByID := make(map[string]*task.Task)
	for _, t := range remoteTasks {
		// Skip archived tasks
		if !archivedByID[t.ID] {
			remoteByID[t.ID] = t
		}
	}

	// Merge: remote wins for conflicts
	var mergedTasks []*task.Task

	// Add all remote tasks
	for _, remote := range remoteByID {
		mergedTasks = append(mergedTasks, remote)
	}

	// Push local-only tasks to remote
	for id, local := range localByID {
		if _, exists := remoteByID[id]; !exists {
			// Task exists locally but not remotely - push it
			if err := s.caldav.CreateTask(ctx, local); err != nil {
				fmt.Printf("Warning: failed to push task %s: %v\n", local.Title, err)
			}
			mergedTasks = append(mergedTasks, local)
		}
	}

	s.tasks = mergedTasks
	return s.save()
}

// PushTask pushes a single task to the CalDAV server
func (s *Storage) PushTask(t *task.Task) error {
	if s.caldav == nil {
		return nil // No sync configured
	}

	ctx := context.Background()
	return s.caldav.CreateTask(ctx, t)
}

// DeleteRemoteTask deletes a task from the CalDAV server
func (s *Storage) DeleteRemoteTask(t *task.Task) error {
	if s.caldav == nil {
		return nil
	}

	ctx := context.Background()
	return s.caldav.DeleteTask(ctx, t)
}

// AddTaskWithSync adds a task and optionally syncs to CalDAV
func (s *Storage) AddTaskWithSync(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = append(s.tasks, t)

	if err := s.save(); err != nil {
		return err
	}

	// Push to CalDAV if sync is enabled
	if s.caldav != nil {
		ctx := context.Background()
		if err := s.caldav.CreateTask(ctx, t); err != nil {
			return fmt.Errorf("failed to sync task: %w", err)
		}
	}

	return nil
}

// ToggleCompleteWithSync toggles completion and syncs
func (s *Storage) ToggleCompleteWithSync(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var targetTask *task.Task
	for _, t := range s.tasks {
		if t.ID == id {
			t.ToggleComplete()
			targetTask = t
			break
		}
	}

	if err := s.save(); err != nil {
		return err
	}

	// Sync to CalDAV
	if targetTask != nil && s.caldav != nil {
		ctx := context.Background()
		if err := s.caldav.UpdateTask(ctx, targetTask); err != nil {
			return fmt.Errorf("failed to sync task: %w", err)
		}
	}

	return nil
}

// UpdateTaskWithSync updates a task and syncs to CalDAV if needed
func (s *Storage) UpdateTaskWithSync(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t.UpdatedAt = time.Now()

	for i, existing := range s.tasks {
		if existing.ID == t.ID {
			s.tasks[i] = t
			if err := s.save(); err != nil {
				return err
			}

			// Sync to CalDAV
			if s.caldav != nil {
				ctx := context.Background()
				if err := s.caldav.UpdateTask(ctx, t); err != nil {
					return fmt.Errorf("failed to sync task: %w", err)
				}
			}

			return nil
		}
	}
	return nil
}

// DeleteTaskWithSync deletes a task and removes from CalDAV
func (s *Storage) DeleteTaskWithSync(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var targetTask *task.Task
	for i, t := range s.tasks {
		if t.ID == id {
			targetTask = t
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			break
		}
	}

	if err := s.save(); err != nil {
		return err
	}

	// Delete from CalDAV
	if targetTask != nil && s.caldav != nil {
		ctx := context.Background()
		if err := s.caldav.DeleteTask(ctx, targetTask); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to delete remote task: %v\n", err)
		}
	}

	return nil
}

// ArchiveTask manually archives a single task by ID (only if completed)
// This is a local operation - the task remains on the server for other clients
func (s *Storage) ArchiveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			if !t.Completed {
				return fmt.Errorf("cannot archive incomplete task")
			}

			t.Archived = true
			s.archived = append(s.archived, t)
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("task not found")
}

// ArchiveAllCompletedTasks archives all completed tasks
// This is a local operation - tasks remain on the server for other clients
func (s *Storage) ArchiveAllCompletedTasks() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var active []*task.Task
	count := 0
	for _, t := range s.tasks {
		if t.Completed {
			t.Archived = true
			s.archived = append(s.archived, t)
			count++
		} else {
			active = append(active, t)
		}
	}
	s.tasks = active
	if err := s.save(); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Storage) LoadIssues() error {
	fmt.Println("Loading Issues")

	if !s.cfg.GitHub.Enabled {
		return fmt.Errorf("GitHub integration not enabled in config")
	}

	if s.cfg.GitHub.Username == "" || s.cfg.GitHub.Token == "" {
		return fmt.Errorf("GitHub username or token not configured")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	issues, err := github.FetchIssues(s.cfg.GitHub.Username, s.cfg.GitHub.Token)
	if err != nil {
		return err
	}
	s.issues = issues
	return s.save()
}

func (s *Storage) GetIssues() []*github.Issue {
	issues := make([]*github.Issue, len(s.issues))
	copy(issues, s.issues)
	return issues
}

// GetMyIssues returns issues assigned to the configured GitHub user
func (s *Storage) GetMyIssues() []*github.Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var myIssues []*github.Issue
	for _, issue := range s.issues {
		if issue.Assignee == s.cfg.GitHub.Username {
			myIssues = append(myIssues, issue)
		}
	}
	return myIssues
}

// GetOpenIssues returns all open issues
func (s *Storage) GetOpenIssues() []*github.Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var openIssues []*github.Issue
	for _, issue := range s.issues {
		if issue.State == "open" {
			openIssues = append(openIssues, issue)
		}
	}
	return openIssues
}

// GetMyOpenIssues returns open issues assigned to me
func (s *Storage) GetMyOpenIssues() []*github.Issue {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var myOpenIssues []*github.Issue
	for _, issue := range s.issues {
		if issue.State == "open" && issue.Assignee == s.cfg.GitHub.Username {
			myOpenIssues = append(myOpenIssues, issue)
		}
	}
	return myOpenIssues
}
