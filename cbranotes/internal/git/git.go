package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func Clone(repoURL, destPath string) error {
	cmd := exec.Command("git", "clone", repoURL, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s\n%s", err, string(output))
	}
	return nil
}

func IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func Pull(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s\n%s", err, string(output))
	}
	return nil
}

func HasChanges(path string) (bool, error) {
	cmd := exec.Command("git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %s", err)
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func CommitAll(path string) error {
	// Stage all changes
	addCmd := exec.Command("git", "-C", path, "add", "-A")
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s\n%s", err, string(output))
	}

	// Commit with timestamp
	msg := fmt.Sprintf("sync: %s", time.Now().Format("2006-01-02 15:04:05"))
	commitCmd := exec.Command("git", "-C", path, "commit", "-m", msg)
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		// Check if it's just "nothing to commit"
		if strings.Contains(string(output), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %s\n%s", err, string(output))
	}
	return nil
}

func Push(path string) error {
	cmd := exec.Command("git", "-C", path, "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %s\n%s", err, string(output))
	}
	return nil
}

// SyncStatus contains information about unpushed/unpulled changes
type SyncStatus struct {
	LocalChanges    []string // Uncommitted local changes
	UnpushedCommits []string // Commits not pushed to remote
	UnpulledCommits []string // Commits not pulled from remote
}

// GetSyncStatus returns the current sync status of the repository
func GetSyncStatus(path string) (*SyncStatus, error) {
	status := &SyncStatus{}

	// Get local uncommitted changes
	statusCmd := exec.Command("git", "-C", path, "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %s", err)
	}
	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				status.LocalChanges = append(status.LocalChanges, line)
			}
		}
	}

	// Fetch to update remote refs (silently)
	fetchCmd := exec.Command("git", "-C", path, "fetch", "--quiet")
	fetchCmd.Run() // Ignore errors, remote might be unavailable

	// Get unpushed commits (local commits not on remote)
	unpushedCmd := exec.Command("git", "-C", path, "log", "@{u}..HEAD", "--oneline")
	unpushedOutput, err := unpushedCmd.Output()
	if err == nil && len(strings.TrimSpace(string(unpushedOutput))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(unpushedOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				status.UnpushedCommits = append(status.UnpushedCommits, line)
			}
		}
	}

	// Get unpulled commits (remote commits not in local)
	unpulledCmd := exec.Command("git", "-C", path, "log", "HEAD..@{u}", "--oneline")
	unpulledOutput, err := unpulledCmd.Output()
	if err == nil && len(strings.TrimSpace(string(unpulledOutput))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(unpulledOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				status.UnpulledCommits = append(status.UnpulledCommits, line)
			}
		}
	}

	return status, nil
}

