# Quick Start Guide - cbrawatch

Get up and running with cbrawatch in 5 minutes.

## 1. Build the App

```bash
cd cbrawatch
go build -o cbrawatch .
```

## 2. (Optional) Install to PATH

```bash
# Move to a directory in your PATH
mv cbrawatch ~/.local/bin/

# Or add to PATH temporarily
export PATH=$PATH:$(pwd)
```

## 3. Run It!

```bash
./cbrawatch
# or if installed:
cbrawatch
```

On first run, it creates a default config at `~/.config/cbraapps/cbrawatch.toml` that scans `~/Code` for repositories.

## 4. Configure Your Paths

Edit the config file:

```bash
nano ~/.config/cbraapps/cbrawatch.toml
```

Minimal config:

```toml
[[paths]]
path = "~/Code"
scan_depth = 2

# Optional: Add custom names for specific repos
[[paths]]
path = "~/Code/my-important-project"
name = "Main Project"  # Shows in list instead of path
scan_depth = 0

max_depth = 2
show_hidden = false
default_commit_message = "Quick update"
refresh_interval_seconds = 0
```

## 5. Use the App

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate repos |
| `/` | Start filtering |
| `Esc` | Clear filter |
| `p` | Quick push (default message) |
| `P` | Push with custom message |
| `u` | Pull from remote |
| `r` | Refresh status |
| `q` | Quit |

### Status Colors

- 🟢 **Green** = Clean (nothing to do)
- 🟡 **Amber** = Changes to commit/push
- 🔵 **Blue** = Updates available (pull needed)
- 🔴 **Red** = Error

### UI Features

- **Cursor Indicator** - Arrow (▶) shows which repository is selected
- **Spinner** - Shows while scanning or performing git operations
- **List Component** - Smooth scrolling for many repositories
- **Custom Names** - Optional display names for repositories (instead of paths)
- **Filtering** - Press `/` and type to filter repos by path
- **Help Bar** - Bottom of screen shows available keyboard shortcuts

## Common Workflows

### Quick Daily Commits

1. Navigate to repo with `↑`/`↓`
2. Press `p` for instant commit+push with default message
3. Done! ✓

### Professional Commits

1. Navigate to repo
2. Press `P` (uppercase)
3. Type your commit message
4. Press Enter
5. Done! ✓

### Pull Updates

1. Navigate to repo with blue indicator (🔵)
2. Press `u` to pull
3. Done! ✓

## Troubleshooting

### "No repositories found"

- Check your config paths exist
- Verify paths contain `.git` directories
- Increase `scan_depth` if repos are nested deeper

### "git push failed: no upstream branch"

Your repo doesn't have a remote branch set. Run in the repo:

```bash
git push -u origin main  # or your branch name
```

### Config not working?

Make sure it's at the right location:

```bash
cat ~/.config/cbraapps/cbrawatch.toml
```

### Want to watch specific repos only?

Use `scan_depth = 0` for exact path checking:

```toml
[[paths]]
path = "~/Code/myproject"
name = "My Important Project"  # Optional custom name
scan_depth = 0  # Only check this exact directory
```

## Tips

- Set `scan_depth = 1` for directories with multiple project folders
- Use `scan_depth = 2` or `-1` for nested project structures
- Edit `default_commit_message` for your preferred quick commit style
- Press `r` after manual git operations to refresh the display
- Use `/` to filter repositories when you have many - great for large workspaces
- Add custom `name` fields to make important repos easier to spot
- The app shows a spinner while loading - no need to wonder if it's working!
- List automatically scrolls and paginates for 100+ repositories
- Look for the ▶ arrow to see which repository is currently selected

## Troubleshooting Custom Names

### Custom name not showing?

Custom names **only work** when:
1. The configured `path` itself is a git repository (has a `.git` directory)
2. You set `scan_depth = 0` for that path

**Common mistake:**

```toml
# This WON'T work if notes/ contains multiple git repos inside
[[paths]]
path = "~/Documents/notes"
scan_depth = -1  # ❌ Finds repos in subdirectories, name doesn't apply to them
name = "My Notes"
```

**Fix: Use scan_depth = 0**

```toml
# This WILL work if ~/Documents/notes/.git exists
[[paths]]
path = "~/Documents/notes"
scan_depth = 0  # ✓ Only checks this exact directory
name = "My Notes"
```

**If you have multiple repos in a directory:**

Add each one individually:

```toml
[[paths]]
path = "~/Documents/notes/personal"
name = "Personal Notes"
scan_depth = 0

[[paths]]
path = "~/Documents/notes/work"
name = "Work Notes"
scan_depth = 0
```

## Next Steps

- Read [README.md](./README.md) for full documentation
- Check [FEATURES.md](./FEATURES.md) for implementation details
- See [config.example.toml](./config.example.toml) for all options

Happy monitoring! 🚀