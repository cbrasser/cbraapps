package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"cbrateach/internal/config"
	"cbrateach/internal/storage"
	"cbrateach/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// Ensure default email template exists
	if err := cfg.EnsureDefaultEmailTemplate(); err != nil {
		log.Fatalf("Failed to create default email template: %v", err)
	}

	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "import":
			handleImport(cfg)
			return
		case "import-school":
			handleImportSchool(cfg)
			return
		case "add-test":
			handleAddTest(cfg)
			return
		case "export-grades":
			handleExportGrades(cfg)
			return
		}
	}

	// Default: run TUI
	model := tui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func handleImport(cfg config.Config) {
	importCmd := flag.NewFlagSet("import", flag.ExitOnError)
	file := importCmd.String("file", "", "Path to CSV file")
	course := importCmd.String("course", "", "Course name to import students into")

	if err := importCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *file == "" {
		log.Fatal("Error: --file flag is required")
	}

	if *course == "" {
		log.Fatal("Error: --course flag is required")
	}

	// Perform import
	store := storage.New(cfg)
	if err := store.ImportStudentsFromCSV(*file, *course); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}

func handleImportSchool(cfg config.Config) {
	importCmd := flag.NewFlagSet("import-school", flag.ExitOnError)
	file := importCmd.String("file", "", "Path to school XLSX file")

	if err := importCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *file == "" {
		log.Fatal("Error: --file flag is required")
	}

	// Perform school import
	store := storage.New(cfg)
	if err := store.ImportCourseFromSchoolXLSX(*file); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}

func handleAddTest(cfg config.Config) {
	addTestCmd := flag.NewFlagSet("add-test", flag.ExitOnError)
	course := addTestCmd.String("course", "", "Course name")
	name := addTestCmd.String("name", "", "Test name")
	topic := addTestCmd.String("topic", "", "Test topic")
	points := addTestCmd.String("points", "", "Path to CSV file with points")
	weight := addTestCmd.Float64("test-weight", 1.0, "Weight for final grade calculation (default 1.0)")

	if err := addTestCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *course == "" || *name == "" || *topic == "" || *points == "" {
		log.Fatal("Error: --course, --name, --topic, and --points flags are required")
	}

	// Find course
	store := storage.New(cfg)
	courses, err := store.LoadCourses()
	if err != nil {
		log.Fatalf("Failed to load courses: %v", err)
	}

	var courseID string
	var courseName string
	for _, c := range courses {
		if c.Name == *course {
			courseID = c.ID
			courseName = c.Name
			break
		}
	}

	if courseID == "" {
		log.Fatalf("Course not found: %s", *course)
	}

	// Import test
	if err := store.ImportTestFromCSV(*points, courseID, courseName, *name, *topic, *weight); err != nil {
		log.Fatalf("Failed to import test: %v", err)
	}
}

func handleExportGrades(cfg config.Config) {
	exportCmd := flag.NewFlagSet("export-grades", flag.ExitOnError)
	course := exportCmd.String("course", "", "Course name")
	output := exportCmd.String("output", "", "Output CSV file path")

	if err := exportCmd.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *course == "" || *output == "" {
		log.Fatal("Error: --course and --output flags are required")
	}

	// Find course
	store := storage.New(cfg)
	courses, err := store.LoadCourses()
	if err != nil {
		log.Fatalf("Failed to load courses: %v", err)
	}

	var courseID string
	for _, c := range courses {
		if c.Name == *course {
			courseID = c.ID
			break
		}
	}

	if courseID == "" {
		log.Fatalf("Course not found: %s", *course)
	}

	// Export grades
	if err := store.ExportGrades(courseID, *output); err != nil {
		log.Fatalf("Failed to export grades: %v", err)
	}

	fmt.Printf("Grades exported to: %s\n", *output)
}
