# CBRANOTES

Tiny TUI App to sync notes through git. 

## Usage

- `cbranotes sync up` -> Push Changes to upstream, commit message is a timestamp
- `cbranotes sync down` -> Pull from upstream
- `cbranotes sync status`-> See unsynced changes
- `cbranotes editor` -> Open notes dir in simple editor

On first usage, the program will ask the user for folder to sync and upstream URL

## Configuration

The config should be located in `/home/user/.config/cbranotes/config.tml`. The app will create a default config uppon first start. 