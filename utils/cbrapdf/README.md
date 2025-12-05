# Markdown to PDF with Glow Styling

Convert markdown files to beautiful PDFs using [Glow](https://github.com/charmbracelet/glow)'s styling.

## Installation

### Install Dependencies

**On Ubuntu/Debian:**
```bash
sudo apt install pandoc
pip install weasyprint
```

**On macOS:**
```bash
brew install pandoc
pip install weasyprint
```

**On Arch Linux:**
```bash
sudo pacman -S pandoc python-weasyprint
```

### Add to PATH (Optional)

To use `md2pdf` from anywhere:

```bash
# Add to your ~/.bashrc or ~/.zshrc
export PATH="$PATH:/home/cbra/Code/cbrapdf"
```

Or create a symlink:
```bash
sudo ln -s /home/cbra/Code/cbrapdf/md2pdf /usr/local/bin/md2pdf
```

## Usage

```bash
# Convert with dark theme (default)
./md2pdf example.md

# Convert with light theme
./md2pdf example.md output.pdf light

# Auto-generate output filename with light theme
./md2pdf example.md - light

# Show help
./md2pdf --help
```

## Examples

```bash
# Basic conversion (creates example.pdf)
./md2pdf example.md

# Specify output file and theme
./md2pdf example.md my-document.pdf light

# Multiple files
for file in *.md; do
    ./md2pdf "$file" - dark
done
```

## Files

- `md2pdf` - The conversion script
- `styles_dark.css` - Dark theme CSS (from Glow)
- `styles_light.css` - Light theme CSS (from Glow)
- `example.md` - Sample markdown file for testing

## Features

- ✅ Glow's beautiful dark and light themes
- ✅ Syntax highlighting for code blocks
- ✅ Tables, lists, and blockquotes
- ✅ Links and images
- ✅ Task lists with checkboxes
- ✅ Proper heading hierarchy
- ✅ Easy command-line interface

## Credits

Styles converted from [Glow](https://github.com/charmbracelet/glow) and [Glamour](https://github.com/charmbracelet/glamour) by Charm.
