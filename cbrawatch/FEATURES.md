# cbrawatch - Features & Implementation Notes

## Overview
A Git repository monitoring dashboard built with Go, Bubbletea, Bubbles, and Huh.

## Current Features

### ✅ Core Functionality
- **Repository Discovery**: Automatically scans configured paths for Git repositories
- **Status Monitoring**: Real-time status checking for each repository
- **Visual Indicators**: Color-coded status dots (green/amber/blue/red)
- **Cursor Indicator**: Clear arrow (▶) showing selected repository
- **Custom Repo Names**: Optional display names from config instead of paths
- **Interactive Navigation**: Vim-style and arrow key navigation with bubbles list component
- **Built-in Filtering**: Type to filter repositories by path
- **Smooth Scrolling**: Automatic viewport management for large repository lists
- **Git Operations**: Quick access to common Git workflows

### ✅ Git Status Detection
The app detects and displays:
- Unstaged changes (modified files not yet staged)
- Uncommitted changes (staged files not yet committed)
- Unpushed commits (local commits ahead of remote)
- Upstream changes (remote commits behind local)
- Current branch name
- Error states (invalid repos, permission issues, etc.)

### ✅ Git Operations
- **Quick Push (p)**: Add all, commit with default message, and push
- **Push with Message (P)**: Prompts for commit message via Huh form, then add/commit/push
- **Pull (u)**: Pull latest changes from remote
- **Refresh (r)**: Manually refresh all repository statuses

### ✅ Configuration System
TOML-based configuration at `~/.config/cbraapps/cbrawatch.toml`:
- Multiple scan paths with individual depth settings
- Optional custom display names for specific repositories
- Global max depth for recursive scanning
- Hidden directory scanning toggle
- Configurable default commit message
- Auto-refresh interval (currently manual only)

### ✅ UI/UX Features
- **Bubbles List Component**: Professional scrollable list with pagination
- **Bubbles Help Component**: Context-aware keyboard shortcuts with proper key binding system
- **Cursor Indicator**: Arrow (▶) showing selected repository with proper spacing alignment
- **Lipgloss Styling**: Beautiful, consistent color scheme
- **Custom Delegate**: Two-line items showing path/branch and status details
- **Custom Display Names**: Shows configured names instead of paths for easier identification
- **Status Summary**: Each repo shows path, branch, and status details with color-coded descriptions
- **Message Box**: Feedback for operations (success/error/info)
- **Processing States**: Visual feedback during long operations with spinner
- **Alt Screen Mode**: Non-destructive terminal usage
- **Filter Mode**: Built-in filtering for finding repositories quickly

## Implementation Details

### Architecture
```
main.go                          # Entry point
├── internal/config/
│   └── config.go               # TOML config loading/saving
├── internal/scanner/
│   └── scanner.go              # Recursive git repo discovery
├── internal/git/
│   └── status.go               # Git operations and status checks
└── internal/tui/
    ├── tui.go                  # Bubbletea Model/Update/View with bubbles components
    └── styles.go               # Lipgloss styling for list items and UI
```

### Bubbletea Pattern
- **Model**: Holds repos, config, list.Model, help.Model, view state
- **Update**: Handles key events and async operation results
- **View**: Renders bubbles list or commit form
- **Commands**: Async git operations (scan, pull, push, etc.)
- **Delegates**: Custom item delegate for rendering repository items
- **Key Bindings**: Structured key.Binding system for help integration

### Message Types
- `scanCompleteMsg`: Returns discovered repositories
- `gitOperationMsg`: Returns success/failure of git operations
- `commitFormCompleteMsg`: Returns user-entered commit message

### View States
- `viewList`: Main repository list view with bubbles list component
- `viewCommitForm`: Huh form for commit message input

### Bubbles Components Used
- **list.Model**: Professional list component with filtering, scrolling, and pagination
- **help.Model**: Context-aware help display with key bindings
- **key.Binding**: Structured keyboard shortcut definitions

### List Item Implementation
- **repoItem**: Implements list.Item and list.DefaultItem interfaces
- **repoDelegate**: Custom delegate for two-line rendering (title + description)
- **Title**: Shows status indicator, path, and branch name
- **Description**: Shows detailed status with color-coded styling
- **FilterValue**: Enables filtering by repository path

## Technical Decisions

### Why Separate Quick Push & Custom Message Push?
- **Efficiency**: Quick push (p) for rapid updates on personal projects
- **Flexibility**: Custom message (P) for professional commits requiring documentation
- **User Choice**: Different workflows for different contexts

### Why Manual Refresh Default?
- **Performance**: Avoids constant git operations on many repos
- **Battery**: No background polling on laptops
- **Flexibility**: Config option exists for auto-refresh if desired

### Why Color-Coded Dots?
- **Speed**: Instant visual scanning of multiple repos
- **Clarity**: Universal color language (green=good, red=bad, yellow=attention)
- **Space**: Compact representation fits many repos on screen

### Error Handling Improvements
- Validates repos before operations
- Provides specific error messages (no upstream, nothing to commit, etc.)
- Wraps errors with context for better debugging

## Future Enhancement Ideas

### Potential Features
- [x] Repository filtering/searching (implemented with bubbles list)
- [ ] Multi-repo operations (select multiple with checkboxes)
- [ ] Git fetch before status check (optional)
- [ ] Stash operations
- [ ] Branch switching
- [ ] View git log for selected repo
- [ ] Open repo in external terminal
- [ ] Custom git commands from config
- [ ] Repository grouping/tagging
- [ ] Persistent selection across refreshes
- [ ] Export status report (JSON/text)
- [ ] Desktop notifications for upstream changes
- [ ] Integration with GitHub/GitLab APIs for PR status
- [ ] Diff preview for uncommitted changes
- [ ] Conflict detection and highlighting

### Performance Optimizations
- [x] Viewport optimization (bubbles list handles large lists efficiently)
- [ ] Parallel status checking with worker pool
- [ ] Incremental refresh (only changed repos)
- [ ] Status caching with TTL
- [ ] Background refresh with debouncing

### UX Improvements
- [x] Viewport/scrolling for many repositories (bubbles list)
- [x] Built-in filtering (bubbles list)
- [ ] Sort options (by status, path, last commit)
- [ ] Collapse/expand repo details
- [ ] Split view: list + detail pane
- [ ] Themes/color scheme customization
- [ ] Progress bar for multi-repo operations
- [ ] Toggle full/compact help view

## Known Limitations

1. **No Async Status Checks**: Currently scans sequentially (could be slow with many repos)
2. ~~**No Pagination**: List could overflow screen with 100+ repos~~ **FIXED** - Bubbles list handles this
3. **Simple Git Detection**: Only checks for `.git` directory (doesn't handle worktrees)
4. **No Conflict Detection**: Doesn't warn about merge conflicts before pull
5. **English Only**: No i18n support

## Development Notes

### Adding New Git Operations
1. Add function to `internal/git/status.go`
2. Create command function in `internal/tui/tui.go`
3. Add key.Binding to keyMap struct with proper help text
4. Add key match case in `updateListView()`
5. Help system automatically updates (no manual changes needed)
6. Add to README keyboard shortcuts section

### Styling Guide
Use existing Lipgloss styles from `styles.go`:
- `cleanColor/warningColor/dangerColor/infoColor` for semantic meaning
- `primaryColor` for highlights and accents
- `mutedColor` for secondary information
- `listItemStyle/selectedItemStyle` for list rendering
- `listItemDescStyle/selectedItemDescStyle` for descriptions
- Keep consistent with other cbraapps and bubbles conventions

### Testing Locally
```bash
# Build
cd cbrawatch
go build -o cbrawatch .

# Run with default config
./cbrawatch

# Test with custom paths
# Edit ~/.config/cbraapps/cbrawatch.toml first
./cbrawatch
```

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework (Elm architecture)
- `github.com/charmbracelet/bubbles` - UI components (list, help, key, spinner)
- `github.com/charmbracelet/huh` - Form components
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/pelletier/go-toml/v2` - TOML parsing

## Credits

Part of the cbraapps collection - personal TUI productivity tools.
Built with Go and the excellent Charm libraries.