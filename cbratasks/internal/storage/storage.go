package storage

import (
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
)

type Storage struct {
	tasks      []*task.Task
	archived   []*task.Task
	dataDir    string
	mu         sync.RWMutex
	caldav     *caldav.Client
	cfg        *config.Config
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
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	s := &Storage{
		dataDir: dataDir,
		cfg:     cfg,
	}

	// Initialize CalDAV client if sync is enabled
	if cfg.Sync.Enabled && cfg.Sync.URL != "" {
		s.caldav = caldav.NewClient(cfg.Sync.URL, cfg.Sync.Username, cfg.Sync.Password)
	}

	if err := s.load(); err != nil {
		return nil, err
	}

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
	if err := os.WriteFile(s.tasksFile(), data, 0644); err != nil {
		return err
	}

	// Save archived tasks
	archiveData, err := json.MarshalIndent(s.archived, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.archiveFile(), archiveData, 0644)
}

// archiveOldTasks moves completed tasks older than 24h to archive
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

	// Ensure collection exists
	if err := s.caldav.EnsureCollection(); err != nil {
		return fmt.Errorf("failed to ensure collection: %w", err)
	}

	// Pull remote tasks
	remoteTasks, err := s.caldav.GetAllTasks()
	if err != nil {
		return fmt.Errorf("failed to fetch remote tasks: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Build map of archived task IDs to filter them out
	archivedByID := make(map[string]bool)
	for _, t := range s.archived {
		archivedByID[t.ID] = true
	}

	// Build maps for comparison
	localByID := make(map[string]*task.Task)
	for _, t := range s.tasks {
		if t.ListName == "radicale" {
			localByID[t.ID] = t
		}
	}

	remoteByID := make(map[string]*task.Task)
	for _, t := range remoteTasks {
		// Skip tasks that are in our archive
		if !archivedByID[t.ID] {
			remoteByID[t.ID] = t
		}
	}

	// Merge: remote wins for conflicts, but we push local-only tasks
	var mergedTasks []*task.Task

	// Keep local-only tasks (non-radicale)
	for _, t := range s.tasks {
		if t.ListName != "radicale" {
			mergedTasks = append(mergedTasks, t)
		}
	}

	// Process remote tasks (filtered to exclude archived)
	for _, remote := range remoteByID {
		mergedTasks = append(mergedTasks, remote)
	}

	// Push local radicale tasks that don't exist remotely
	for id, local := range localByID {
		if _, exists := remoteByID[id]; !exists {
			// Task exists locally but not remotely - push it
			if err := s.caldav.CreateTask(local); err != nil {
				// Log but continue
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

	if t.ListName != "radicale" {
		return nil // Only push radicale tasks
	}

	return s.caldav.CreateTask(t)
}

// DeleteRemoteTask deletes a task from the CalDAV server
func (s *Storage) DeleteRemoteTask(id string) error {
	if s.caldav == nil {
		return nil
	}

	return s.caldav.DeleteTask(id)
}

// AddTaskWithSync adds a task and optionally syncs to CalDAV
func (s *Storage) AddTaskWithSync(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks = append(s.tasks, t)

	if err := s.save(); err != nil {
		return err
	}

	// Push to CalDAV if it's a radicale task
	if t.ListName == "radicale" && s.caldav != nil {
		if err := s.caldav.EnsureCollection(); err != nil {
			return fmt.Errorf("failed to ensure collection: %w", err)
		}
		if err := s.caldav.CreateTask(t); err != nil {
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
	if targetTask != nil && targetTask.ListName == "radicale" && s.caldav != nil {
		if err := s.caldav.UpdateTask(targetTask); err != nil {
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
			if t.ListName == "radicale" && s.caldav != nil {
				if err := s.caldav.UpdateTask(t); err != nil {
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
	if targetTask != nil && targetTask.ListName == "radicale" && s.caldav != nil {
		if err := s.caldav.DeleteTask(id); err != nil {
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

