# cbratasks

A minimal terminal-based task manager with local storage and optional CalDAV sync (Radicale).

## Features

- **Simple task management** - Add, complete, delete tasks with markdown-style checkboxes
- **Notes** - Attach text notes to any task (syncs with CalDAV as DESCRIPTION)
- **Tags with colors** - Categorize tasks with customizable colored tags
- **Due dates** - Flexible date input (+1d, tomorrow, nextweek, etc.)
- **CalDAV sync** - Sync with Radicale or other CalDAV servers
- **Auto-archive** - Completed tasks auto-archive after 24 hours
- **Fuzzy search** - Quickly find tasks
- **Customizable hotkeys** - Configure keybindings in config file

## Installation

### From source

```bash
# Clone the repository
git clone https://github.com/yourusername/cbraapps.git
cd cbraapps/cbratasks

# Build
go build -o cbratasks .

# Move to your PATH
mv cbratasks ~/.local/bin/
# or
sudo mv cbratasks /usr/local/bin/
```

### Requirements

- Go 1.21 or later

## Usage

### TUI Mode (default)

Simply run `cbratasks` to open the interactive terminal UI:

```bash
cbratasks
```

#### Keybindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate tasks |
| `x` | Toggle complete |
| `a` | Add new task |
| `n` | Edit note for current task |
| `tab` | View note (if task has one) |
| `d` | Delete task |
| `/` | Search tasks |
| `s` | Manual sync with CalDAV |
| `q` | Quit |

### Command Line

#### Add tasks

```bash
# Simple task
cbratasks add "Buy groceries"

# With due date
cbratasks add "Meeting with John" --due tomorrow
cbratasks add "Submit report" --due +3d
cbratasks add "Quarterly review" --due 25-12-2024

# With tags
cbratasks add "Fix login bug" --tag work --tag urgent

# With a note
cbratasks add "Call mom" --note "Ask about birthday plans"

# To specific list (local or radicale)
cbratasks add "Sync this task" --list radicale
```

#### Due date formats

| Format | Example | Description |
|--------|---------|-------------|
| `+Nd` | `+1d`, `+3d` | N days from now |
| `+Nw` | `+1w`, `+2w` | N weeks from now |
| `+Nm` | `+1m` | N months from now |
| `today` | | End of today |
| `tomorrow` | | End of tomorrow |
| `nextweek` | | Next Monday |
| `DD-MM-YYYY` | `25-12-2024` | Specific date |
| `YYYY-MM-DD` | `2024-12-25` | ISO format |

#### List tasks

```bash
cbratasks list
```

#### Tasks due today

Get a list of tasks due today (useful for scripts/integrations):

```bash
cbratasks today
```

Output format:
```
- Task title [tags] (task-id)
```

#### View archived tasks

```bash
cbratasks archive
```

#### Sync with CalDAV

```bash
cbratasks sync
```

## Configuration

Configuration is stored in `~/.config/cbratasks/config.toml`. It's auto-generated on first run.

```toml
# Default task list: "local" or "radicale"
default_list = "local"

[sync]
enabled = false
url = "https://radicale.example.com"
username = ""
password = ""

[tags]
work = "#FF6B6B"
home = "#4ECDC4"
personal = "#95E1D3"
urgent = "#F38181"
shopping = "#AA96DA"

[hotkeys]
mark_complete = "x"
delete = "d"
edit_note = "n"
view_note = "tab"
add_task = "a"
search = "/"
quit = "q"
```

### CalDAV Sync (Radicale)

To enable sync with a Radicale server:

1. Edit `~/.config/cbratasks/config.toml`:

```toml
[sync]
enabled = true
url = "https://your-radicale-server.com"
username = "your_username"
password = "your_password"
```

2. Set default list to radicale (optional):

```toml
default_list = "radicale"
```

3. Run sync:

```bash
cbratasks sync
```

A `cbratasks` collection will be automatically created on the server if it doesn't exist.

### Notes & CalDAV

Notes are synced with CalDAV using the standard `DESCRIPTION` field in VTODO items. This means notes will appear in other CalDAV-compatible apps that display task descriptions.

### Custom Tag Colors

Add your own tags with custom colors in the config:

```toml
[tags]
work = "#FF6B6B"
home = "#4ECDC4"
fitness = "#FFE66D"
reading = "#6BCB77"
```

Colors are specified in hex format.

## Data Storage

- **Tasks**: `~/.config/cbratasks/data/tasks.json`
- **Archive**: `~/.config/cbratasks/data/archive.json`
- **Config**: `~/.config/cbratasks/config.toml`

## License

MIT
