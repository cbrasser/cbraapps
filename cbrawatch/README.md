# cbrawatch

A terminal dashboard for monitoring Git repositories. Built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Features

- 🔍 **Auto-discover** Git repositories from configured paths
- 📊 **Visual status indicators** - colored dots show repo status at a glance
- 🏷️ **Custom repo names** - Optional custom display names for repositories
- 📋 **List component** - Smooth scrolling and navigation with bubbles list
- 🎯 **Clear cursor indicator** - Arrow (▶) shows selected repository
- 🔎 **Built-in filtering** - Type to filter repositories by path
- ⚡ **Quick actions** via hotkeys:
  - Quick commit & push with default message
  - Commit & push with custom message prompt
  - Pull latest changes
  - Manual refresh
- 🎨 **Beautiful TUI** with colors and status information
- ⌨️ **Context-aware help** - Built-in help system showing available keys
- ⚙️ **Configurable** scanning depth and paths

## Status Indicators

- 🟢 **Green** - Repository is clean (no changes)
- 🟡 **Amber** - Uncommitted or unpushed changes
- 🔵 **Blue** - Upstream changes available (pull needed)
- 🔴 **Red** - Error accessing repository

## Installation

```bash
cd cbrawatch
go build -o cbrawatch .
mv cbrawatch ~/.local/bin/  # or another directory in your PATH
```

## Usage

Simply run the app:

```bash
cbrawatch
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑`/`↓` or `k`/`j` | Navigate through repositories |
| `/` | Start filtering repositories |
| `Esc` | Clear filter |
| `p` | Quick push - add all, commit with default message, and push |
| `P` | Push with message - prompts for commit message, then add all, commit, and push |
| `u` | Pull latest changes from remote |
| `r` | Refresh repository status |
| `q` or `Ctrl+C` | Quit |

**Tip:** When filtering is active, type to search for repositories by path. Press `Esc` to clear the filter.

## Configuration

On first run, cbrawatch creates a config file at:
```
~/.config/cbraapps/cbrawatch.toml
```

### Example Configuration

```toml
# Paths to scan for git repositories
[[paths]]
path = "~/Code"
scan_depth = -1  # -1 = use global max_depth, 0 = exact path only, N = scan N levels deep

[[paths]]
path = "~/Projects"
scan_depth = 2

# Optional: Add custom display names for specific repos
# IMPORTANT: Use scan_depth = 0 when adding custom names
# This ensures the name applies to the exact repo at that path
[[paths]]
path = "~/Code/my-important-project"
name = "Main Project"  # Shows "Main Project" instead of the path
scan_depth = 0  # Must be 0 for custom name to work properly

# Global maximum scanning depth
max_depth = 2

# Show hidden directories when scanning
show_hidden = false

# Default commit message for quick push (lowercase 'p')
default_commit_message = "Quick update"

# Auto-refresh interval in seconds (0 = manual refresh only)
refresh_interval_seconds = 0
```

### Configuration Options

- **`paths`** - Array of paths to scan for repositories
  - `path` - Directory to scan (supports `~` for home directory)
  - `name` - Optional custom display name for the repository in the list
    - **Note:** Custom names only apply when the configured path itself is a git repository
    - If `scan_depth > 0`, the name applies to repos found at the exact path, not subdirectories
    - For best results with custom names, use `scan_depth = 0` for specific repos
  - `scan_depth` - How deep to scan (-1 uses global `max_depth`, 0 checks exact path only)
- **`max_depth`** - Global maximum depth for repository scanning
- **`show_hidden`** - Whether to scan hidden directories (starting with `.`)
- **`default_commit_message`** - Message used for quick push (`p` key)
- **`refresh_interval_seconds`** - Auto-refresh interval (0 disables auto-refresh)

## Repository Status Information

For each repository, cbrawatch shows:
- **Cursor indicator** (▶) for the selected repository
- **Status indicator** (colored dot)
- **Repository path** (or custom name if configured)
- **Current branch** name in brackets
- **Status summary** with details:
  - `unstaged` - Files modified but not added
  - `uncommitted` - Files staged but not committed
  - `↑N` - N commits ahead of remote (unpushed)
  - `↓N` - N commits behind remote (need to pull)
  - `clean` - No changes, in sync with remote

## How It Works

1. On startup, cbrawatch scans all configured paths for Git repositories
2. For each repository found, it checks:
   - Unstaged changes (`git status`)
   - Uncommitted changes (staged files)
   - Unpushed commits (ahead of remote)
   - Available updates (behind remote)
   - Current branch name
3. Displays all repositories in a scrollable, filterable list with visual status indicators
4. Uses Charm's bubbles components for a polished, interactive experience
5. Allows you to perform common Git operations with single keystrokes

## Requirements

- Go 1.21 or later
- Git installed and available in PATH

## Tips

- Use `scan_depth = 0` for paths that contain a single repository
- Use `scan_depth = 1` or `2` for directories containing multiple projects
- The quick push (`p`) is great for personal projects with simple commit messages
- Use custom message push (`P`) for more descriptive commits
- Status indicators help you quickly spot repositories needing attention
- Use the filter (`/`) to quickly find specific repositories in large lists
- The list component handles scrolling automatically for many repositories