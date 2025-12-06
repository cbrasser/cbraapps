So here's the plan for this repository:

- it will be a teacher's utility application ✓
data model:
- a course has number of students, each with name and e-mail. Student can have a note attached but does not have to. course has a name and a subject. Each course has a markdown note attached by default. each course has a time and weekday and a room. a course has a a current topic that is currently discussed in class. this will change over the school year. ✓
views:
- when starting the app the user sees a list view of courses. they are presented with some shortcuts: send e-mail (through <https://github.com/charmbracelet/pop> ) to all students of selected class, open classbook, open course-note, fill out "after-class review", maybe more to come. ✓
- The classbook is the detail view of a course. in here, the teacher sees: list of student names in one pane, course details in another.. there are shortcuts for sending an e-mail to the selected student, editing the note of the selected student. ✓
The "after-class-review" is a form where the teacher fills in a short review ofa session. they are prompted to fill in: Title (default to course name and date), review (textarea), students who stood out positively and negatively (expandable list, add +/- after the name to mark the direction), topic (default to current course topic). ✓

data storage:

- configure app through TOML file in .config/cbraapps/cbrateach.TOML ✓
- folder for course notes in MD can be configured. ✓
- folder for the other data can be configured separately ✓
- all will be stored locally to respect privacy laws ✓

Build everything using GO and libraries associated with bubbleTea:

- Bubbles for TUI components ✓
- glamour for MD styling if needed (not needed)
- lipgloss for styles ✓
- huh for prompts/forms ✓

Additional features:

- CSV/XLSX import of students: `cbrateach import --file <path> --course <course_name>` ✓
  - Supports both .csv and .xlsx file formats
  - Auto-detects file type by extension
- School-specific import: `cbrateach import-school --file <path>` ✓
  - Handles Gymneufeld format (Klasse name, Vorname/Nachname columns)
  - Auto-creates courses from file
  - Combines first/last names
- Sender email configuration in TOML config file ✓
- Automatic sender email in pop email commands ✓
