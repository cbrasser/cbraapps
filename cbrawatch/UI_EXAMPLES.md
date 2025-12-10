# cbrawatch - UI Examples

This document shows what the UI looks like in different states.

## Initial Loading State

When you first launch the app or press `r` to refresh with no repos loaded yet:

```
🔍 Git Repository Monitor

  ⠋ Scanning repositories...

  Please wait while the operation completes.
```

## Main List View

After repositories are loaded:

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      clean

    ● ~/Code/project2 [dev]
      uncommitted, ↑2

    ● ~/Code/website [main]
      unstaged

    ● Main Project [feature/new-ui]
      ↓3

    ● ~/Work/client-site [main]
      unstaged, uncommitted, ↑1

✓ Found 5 repositories

p quick push • P push w/ message • u pull • r refresh • q quit
```

### Visual Elements Explained

- **▶** - Cursor indicator (purple) showing which repo is selected
- **●** - Status indicator (colored dot):
  - 🟢 Green = Clean, no changes
  - 🟡 Amber = Uncommitted or unpushed changes
  - 🔵 Blue = Upstream changes available
  - 🔴 Red = Error state
- **Path or Custom Name** - Shows configured `name` field or repo path
- **[branch]** - Current Git branch in brackets
- **Status line** - Detailed status with color coding matching the dot

### Status Colors in Detail

```
▶ ● ~/Code/clean-repo [main]
    clean                          ← Green text (no issues)

  ● ~/Code/dirty-repo [dev]
    uncommitted, ↑2                ← Amber/yellow text (needs attention)

  ● ~/Code/behind-repo [main]
    ↓3                             ← Blue text (upstream changes)

  ● ~/Code/broken-repo [main]
    error                          ← Red text (problem accessing repo)
```

## Filtering Mode

Press `/` to start filtering:

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/website [main]
      unstaged

    ● ~/Work/client-site [main]
      unstaged, uncommitted, ↑1

Filter: site_

p quick push • P push w/ message • u pull • r refresh • q quit
```

Type to search by path or custom name. Press `Esc` to clear the filter.

## Refreshing State (with existing repos)

When refreshing while repos are already displayed:

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      clean

    ● ~/Code/project2 [dev]
      uncommitted, ↑2

    ● ~/Code/website [main]
      unstaged

  ⠙ Refreshing repositories...

p quick push • P push w/ message • u pull • r refresh • q quit
```

## During Git Operations

When performing push operation:

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      clean

    ● ~/Code/project2 [dev]
      uncommitted, ↑2

  ⠹ Pushing changes...

(keyboard shortcuts temporarily disabled)
```

When performing pull operation:

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      clean

    ● ~/Code/project2 [dev]
      uncommitted, ↑2

  ⠹ Pulling changes...

(keyboard shortcuts temporarily disabled)
```

## After Successful Operation

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      clean

    ● ~/Code/project2 [dev]
      clean

✓ add/commit/push completed successfully

p quick push • P push w/ message • u pull • r refresh • q quit
```

## After Failed Operation

```
🔍 Git Repository Monitor

  ▶ ● ~/Code/project1 [main]
      unstaged, uncommitted

✗ push failed: no upstream branch configured

p quick push • P push w/ message • u pull • r refresh • q quit
```

## Commit Message Form (Press P)

When you press `P` for push with custom message:

```
📝 Commit Message

  Commit Message
  > Fix navigation bug in header___________

  • Submit: enter • Abort: esc
```

Type your commit message and press Enter to commit and push.

## Custom Name Examples

Configuration:

```toml
[[paths]]
path = "~/Code/very-long-complicated-project-name"
name = "Main Project"
scan_depth = 0

[[paths]]
path = "~/Work/client-deliverable-2024"
name = "🚀 Client Site"
scan_depth = 0
```

Display in list:

```
  ▶ ● Main Project [main]               ← Shows custom name
      clean

    ● 🚀 Client Site [dev]               ← You can use emojis!
      uncommitted, ↑2
```

This makes it much easier to identify important repositories at a glance!

## Empty State

When no repositories are found:

```
🔍 Git Repository Monitor

No repositories found. Check your config paths.

r refresh • q quit
```

## Many Repositories (Scrolling)

When you have many repos (the list auto-scrolls):

```
🔍 Git Repository Monitor

    ● ~/Code/project8 [main]
      clean

  ▶ ● ~/Code/project9 [dev]              ← Currently selected
      uncommitted

    ● ~/Code/project10 [main]
      ↑1

    ● ~/Code/project11 [hotfix]
      clean

  Items 9/50                             ← Pagination indicator

p quick push • P push w/ message • u pull • r refresh • q quit
```

Use `↑`/`↓` or `j`/`k` to scroll through the list.

## Color Scheme Summary

- **Purple/Pink** (`#8B5CF6`, `#EC4899`) - Primary colors for selection, cursor, titles
- **Green** (`#10B981`) - Clean status, success messages
- **Amber** (`#F59E0B`) - Warning, uncommitted changes
- **Blue** (`#3B82F6`) - Info, upstream changes
- **Red** (`#EF4444`) - Errors, danger
- **Gray** (`#6B7280`) - Muted text, help

## Tips for Best UI Experience

1. **Add custom names** for your most important repositories - makes them stand out
2. **Use filtering** (`/`) when you have many repos - very fast way to find specific ones
3. **Watch the cursor** (▶) - it's always clear which repo will receive your action
4. **Color dots at a glance** - quickly scan for repos needing attention (non-green dots)
5. **Status descriptions** are color-coded to match the dots for consistency