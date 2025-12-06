# cbrateach - Teacher's Utility TUI

A terminal user interface application for teachers to manage courses, students, and class reviews.

## Features

### Data Model
- **Courses**: Name, subject, time, weekday, room, and current topic
- **Students**: Name, email, optional notes, positive/negative marks tracking
- **Reviews**: After-class reviews with title, topic, review text, and student highlights
- **Course Notes**: Markdown notes for each course with automatic review appending
- **Tests & Grades**: Tests with multiple questions, student scores, automatic grade calculation (Swiss system: 1-6)

### Views

#### Main List View
- Browse all courses
- **Keyboard shortcuts:**
  - `↑/k`: Navigate up
  - `↓/j`: Navigate down
  - `enter`: Open classbook for selected course
  - `e`: Send email to all students in course (via [pop](https://github.com/charmbracelet/pop))
  - `n`: Open course note in $EDITOR
  - `r`: Fill out after-class review
  - `a`: Add new course
  - `q`: Quit

#### Classbook View (Detail)
- Two-pane layout: students list and course details
- Shows student marks (+/- indicators)
- **Keyboard shortcuts:**
  - `↑/k`: Navigate students
  - `↓/j`: Navigate students
  - `e`: Send email to selected student
  - `n`: Edit student note
  - `a`: Add new student
  - `t`: View tests for this course
  - `esc`: Return to list view

#### After-Class Review Form
- Title (defaults to course name + date)
- Topic discussed (defaults to current course topic)
- Review notes (optional, appended to course markdown note)
- Select students who stood out (positive/negative tracking)
- Automatically updates student records with marks

#### Test List View
- View all tests for a course (review/confirmed status)
- **Keyboard shortcuts:**
  - `↑/k`, `↓/j`: Navigate
  - `enter`: Open test review
  - `esc`: Back to classbook

#### Test Review View
- Interactive table with Bubbles table component
- Shows: Student names, points per question, total points, grades
- Grade calculation: `(points / (max_points - gifted_points)) * 5 + 1`, rounded to quarters
- Edit individual cells in review mode
- Adjust "gifted points" to make test easier
- Confirm test to finalize grades
- **Keyboard shortcuts:**
  - `↑↓←→`/`hjkl`: Navigate table
  - `e`: Edit selected cell (review mode only)
  - `g`: Edit gifted points (review mode only)
  - `c`: Confirm test (locks grades)
  - `u`: Unconfirm test (back to review)
  - `esc`: Back to test list

## Installation

```bash
# Build
go build -o cbrateach .

# Install to local bin (optional)
cp cbrateach ~/.local/bin/
```

## Configuration

Configuration file: `~/.config/cbraapps/cbrateach.toml`

```toml
data_dir = "/home/user/.config/cbraapps/cbrateach/data"
course_notes_dir = "/home/user/.config/cbraapps/cbrateach/notes"
reviews_dir = "/home/user/.config/cbraapps/cbrateach/reviews"
sender_email = "teacher@example.com"
```

Auto-generated on first run with default paths.

**Important:** Update `sender_email` with your actual email address for automatic email sending via `pop`.

## Data Storage

All data is stored locally to respect privacy laws:

- **Courses**: `~/.config/cbraapps/cbrateach/data/courses.json`
- **Course Notes**: `~/.config/cbraapps/cbrateach/notes/*.md`
- **Reviews**: `~/.config/cbraapps/cbrateach/reviews/*.json`

Course notes automatically include a "### Reviews" section where review texts are appended.

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Huh](https://github.com/charmbracelet/huh) - Forms and prompts
- [go-toml](https://github.com/pelletier/go-toml) - TOML configuration
- [Excelize](https://github.com/xuri/excelize) - Excel file support
- [pop](https://github.com/charmbracelet/pop) - Email integration (optional)

## Usage

```bash
# Run the application
./cbrateach
```

The application will:
1. Create configuration file if it doesn't exist
2. Create necessary directories
3. Load existing courses (or start with empty list)
4. Present the main course list view

### CSV/XLSX Import

Import students from a CSV or Excel file into an existing course:

```bash
# Supports both .csv and .xlsx files
./cbrateach import --file students.csv --course "Math 101"
./cbrateach import --file students.xlsx --course "Math 101"
```

**CSV/XLSX format:** name, email (with optional header row in first row)

**CSV file example:**
```csv
name,email
Alice Johnson,alice.johnson@school.edu
Bob Smith,bob.smith@school.edu
Carol Davis,carol.davis@school.edu
```

**Excel (.xlsx) files:** The first sheet will be used, with the same format (columns: name, email)

The import command will:
- Auto-detect file type (CSV or Excel) by extension
- Skip duplicate students (by name)
- Skip empty rows
- Automatically detect and skip header rows
- Print the number of students imported

### School-Specific Import (Gymneufeld Format)

For school exports in the specific format (Klasse name in row 1, headers in row 3, data starting row 4):

```bash
# Import single course (auto-creates course from file)
./cbrateach import-school --file courses/Report.xlsx

# Bulk import all school files
for file in courses/Report*.xlsx; do
  ./cbrateach import-school --file "$file"
done
```

**School XLSX format:**
- Row 1: Class name (e.g., "Klasse 29Gc")
- Row 2: Empty
- Row 3: Headers (Vorname, Nachname, Email)
- Row 4+: Student data (first name, last name, email)

**Features:**
- Automatically extracts course name from row 1
- Creates course if it doesn't exist
- Combines first and last names into full name
- Skips duplicates
- You can edit course details (subject, time, room, topic) later in the TUI

### Grading & Tests

**Add a test from CSV:**

```bash
./cbrateach add-test --course "29Gc" --name "Test 1" --topic "Algebra" --points test_results.csv
```

**CSV format:** Vorname, Nachname, then question columns with max points in header

```csv
Vorname,Nachname,Q1 (10),Q2 (15),Q3 (20)
John,Doe,8.5,12,18
Jane,Smith,9,14,17
```

**Features:**
- Automatically extracts max points from headers: `Q1 (10)` → 10 points
- Test starts in "review" mode
- View/edit in TUI (press `t` in classbook view)
- Edit individual scores or gifted points
- Confirm when ready to finalize

**Export final grades:**

```bash
./cbrateach export-grades --course "29Gc" --output grades.csv
```

**Output format:** Vorname, Nachname, Grade (average of all confirmed tests)

### Workflow Example

**Option A: School Import (Recommended for Gymneufeld format)**
1. **Bulk import** all school files: `for file in courses/*.xlsx; do ./cbrateach import-school --file "$file"; done`
2. **Fill in course details** in TUI (subject, time, weekday, room, current topic)
3. **Start teaching!**

**Option B: Manual Setup**
1. **Add a course** (press `a` in list view)
2. **Import students** from CSV: `./cbrateach import --file students.csv --course "Course Name"`
   - Or **add students manually** in the TUI (open classbook with `enter`, then press `a`)
3. **Open course note** (press `n`) to add class planning notes

**Daily Workflow:**
1. **After class**, fill out review (press `r`)
2. **Send emails** to students or entire class as needed
3. **Track student progress** with positive/negative marks

## Architecture

The application follows the internal package pattern:

```
cbrateach/
├── main.go                    # Entry point
├── internal/
│   ├── config/               # Configuration management
│   ├── models/               # Data structures
│   ├── storage/              # Persistence layer
│   └── tui/                  # Bubbletea UI components
│       ├── tui.go           # Main model
│       ├── styles.go        # Lipgloss styles
│       ├── list_view.go     # Course list view
│       ├── classbook_view.go # Course detail view
│       ├── forms.go         # Add/edit forms
│       └── review_form.go   # Review form
├── go.mod
└── README.md
```

## License

Part of the cbraapps monorepo.
