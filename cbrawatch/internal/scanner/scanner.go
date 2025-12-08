package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"cbrawatch/internal/config"
	"cbrawatch/internal/git"
)

func ScanRepositories(cfg *config.Config) []git.RepoStatus {
	var repos []git.RepoStatus
	seen := make(map[string]bool)
	customNames := make(map[string]string) // Map absolute path to custom name

	for _, pathCfg := range cfg.Paths {
		expandedPath := expandPath(pathCfg.Path)

		// Store custom name if provided, normalize the path
		if pathCfg.Name != "" {
			absPath, err := filepath.Abs(expandedPath)
			if err == nil {
				// Clean the path to normalize it (removes trailing slashes, etc.)
				cleanPath := filepath.Clean(absPath)
				customNames[cleanPath] = pathCfg.Name
			}
		}

		// Determine scan depth for this path
		depth := pathCfg.ScanDepth
		if depth == -1 {
			depth = cfg.MaxDepth
		}

		foundRepos := scanPath(expandedPath, depth, cfg.ShowHidden, seen)
		repos = append(repos, foundRepos...)
	}

	// Apply custom names to repos (normalize repo paths for comparison)
	for i := range repos {
		cleanRepoPath := filepath.Clean(repos[i].Path)
		if customName, ok := customNames[cleanRepoPath]; ok {
			repos[i].CustomName = customName
		}
	}

	return repos
}

func scanPath(rootPath string, maxDepth int, showHidden bool, seen map[string]bool) []git.RepoStatus {
	var repos []git.RepoStatus

	// Check if root path itself is a git repo
	if isGitRepo(rootPath) {
		absPath, _ := filepath.Abs(rootPath)
		if !seen[absPath] {
			seen[absPath] = true
			status := git.CheckStatus(absPath)
			status.Path = filepath.Clean(absPath) // Normalize the path
			repos = append(repos, status)
		}
		return repos
	}

	// If maxDepth is 0, only check the exact path
	if maxDepth == 0 {
		return repos
	}

	// Scan subdirectories
	repos = append(repos, scanRecursive(rootPath, maxDepth, 0, showHidden, seen)...)
	return repos
}

func scanRecursive(path string, maxDepth, currentDepth int, showHidden bool, seen map[string]bool) []git.RepoStatus {
	var repos []git.RepoStatus

	if currentDepth > maxDepth {
		return repos
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return repos
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden directories unless configured to show them
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(path, name)

		// Check if this directory is a git repo
		if isGitRepo(fullPath) {
			absPath, _ := filepath.Abs(fullPath)
			if !seen[absPath] {
				seen[absPath] = true
				status := git.CheckStatus(absPath)
				status.Path = filepath.Clean(absPath) // Normalize the path
				repos = append(repos, status)
			}
			// Don't recurse into git repos
			continue
		}

		// Recurse into subdirectories
		if currentDepth < maxDepth {
			repos = append(repos, scanRecursive(fullPath, maxDepth, currentDepth+1, showHidden, seen)...)
		}
	}

	return repos
}

func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[1:])
		}
	}
	return path
}
