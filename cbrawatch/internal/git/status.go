package git

import (
	"fmt"
	"os/exec"
	"strings"
)

type RepoStatus struct {
	Path              string
	HasUnstaged       bool
	HasUncommitted    bool
	HasUnpushed       bool
	HasUpstreamChange bool
	BranchName        string
	Error             string
	AheadBy           int
	BehindBy          int
}

func (r *RepoStatus) IsClean() bool {
	return !r.HasUnstaged && !r.HasUncommitted && !r.HasUnpushed && !r.HasUpstreamChange && r.Error == ""
}

func (r *RepoStatus) StatusSummary() string {
	if r.Error != "" {
		return "error"
	}
	if r.IsClean() {
		return "clean"
	}

	var parts []string
	if r.HasUnstaged {
		parts = append(parts, "unstaged")
	}
	if r.HasUncommitted {
		parts = append(parts, "uncommitted")
	}
	if r.HasUnpushed {
		parts = append(parts, fmt.Sprintf("↑%d", r.AheadBy))
	}
	if r.HasUpstreamChange {
		parts = append(parts, fmt.Sprintf("↓%d", r.BehindBy))
	}

	return strings.Join(parts, ", ")
}

func CheckStatus(repoPath string) RepoStatus {
	status := RepoStatus{
		Path: repoPath,
	}

	// Check if it's a git repo
	if !isGitRepo(repoPath) {
		status.Error = "not a git repository"
		return status
	}

	// Get branch name
	branchCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if output, err := branchCmd.Output(); err == nil {
		status.BranchName = strings.TrimSpace(string(output))
	}

	// Check for unstaged changes
	statusCmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := statusCmd.Output()
	if err != nil {
		status.Error = fmt.Sprintf("git status failed: %v", err)
		return status
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		// Check first two characters for status codes
		if line[0] == ' ' && line[1] != ' ' {
			status.HasUnstaged = true
		}
		if line[0] != ' ' && line[0] != '?' {
			status.HasUncommitted = true
		}
		if line[0] == '?' && line[1] == '?' {
			status.HasUnstaged = true
		}
	}

	// Check for unpushed commits and upstream changes
	// First, try to fetch to get latest remote info (silently)
	fetchCmd := exec.Command("git", "-C", repoPath, "fetch", "--dry-run")
	fetchCmd.Run() // Ignore errors, repo might not have remote

	// Get ahead/behind counts
	revListCmd := exec.Command("git", "-C", repoPath, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if output, err := revListCmd.Output(); err == nil {
		counts := strings.Fields(strings.TrimSpace(string(output)))
		if len(counts) == 2 {
			fmt.Sscanf(counts[0], "%d", &status.AheadBy)
			fmt.Sscanf(counts[1], "%d", &status.BehindBy)

			status.HasUnpushed = status.AheadBy > 0
			status.HasUpstreamChange = status.BehindBy > 0
		}
	}

	return status
}

func isGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func AddAll(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "add", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %v\n%s", err, output)
	}
	return nil
}

func Commit(repoPath, message string) error {
	cmd := exec.Command("git", "-C", repoPath, "commit", "-m", message)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %v\n%s", err, output)
	}
	return nil
}

func Push(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "push")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %v\n%s", err, output)
	}
	return nil
}

func Pull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %v\n%s", err, output)
	}
	return nil
}

func AddCommitPush(repoPath, message string) error {
	if err := AddAll(repoPath); err != nil {
		return err
	}
	if err := Commit(repoPath, message); err != nil {
		return err
	}
	if err := Push(repoPath); err != nil {
		return err
	}
	return nil
}
