# cbraapps

A collection of terminal user interface (TUI) applications built with Go and [Bubbletea](https://github.com/charmbracelet/bubbletea).

## Applications

### [cbrabuild](./cbrabuild/)

Build orchestration tool with interactive TUI for selecting and building Go projects from a YAML configuration file.

### [cbrafetch](./cbrafetch/)

System information display tool showing date, time, and week number.

### [cbratube](./cbratube/)

TUI app and yt-dlp frontend for viewing and managing YouTube channels. Download videos, search across channels, and track what you've watched.

### [cbracal](./cbracal/)

TUI calendar app for syncing and editing local & CalDAV calendars. Supports multiple view modes (daily, weekly, monthly) and Radicale integration.

### [cbranotes](./cbranotes/)

Tiny TUI app to sync notes through git. Simple commands for pushing/pulling changes and editing notes.

### [cbratasks](./cbratasks/)

A minimal terminal-based task manager with local storage and optional CalDAV sync (Radicale). Features tags, due dates, notes, and auto-archiving.

## Building

### Build All Apps

Use the interactive build tool:

```bash
cd cbrabuild
go run main.go
```

### Build Individual Apps

```bash
cd <app-name>
go build -o <app-name> .
```

For example:

```bash
cd cbratasks
go build -o cbratasks .
```

## Installation

After building, move binaries to your PATH:

```bash
mv cbratasks/cbratasks ~/.local/bin/
mv cbranotes/cbranotes ~/.local/bin/
# ... etc
```

Or configure cbrabuild to move them automatically (see `cbrabuild/config.yaml`).

## Requirements

- Go 1.21 or later
- For cbratube: optionally install `yt-dlp` for reliable video downloads
- For cbracal/cbratasks: optional Radicale server for CalDAV sync

## Configuration

All apps use TOML configuration files stored in `~/.config/cbraapps/<appname>/config.toml`.

Default configurations are created automatically on first run.

## Documentation

Individual apps have their own README files with usage instructions.
