package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/jessn-dev/sprout/internal/generator"
)

// sproutTheme tints huh prompts with Sprout's green/purple palette so the
// fast module flow visually matches the main TUI.
func sproutTheme() *huh.Theme {
	green := lipgloss.Color("#22C55E")
	purple := lipgloss.Color("#A855F7")
	blue := lipgloss.Color("#3B82F6")

	t := huh.ThemeBase()
	t.Focused.Title = t.Focused.Title.Foreground(purple).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(lipgloss.Color("#6B7280"))
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(green)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(green)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(green).Foreground(lipgloss.Color("#111827")).Bold(true)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(blue)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(green)
	return t
}

// moduleDepOptions builds the multi-select list from live start.spring.io
// metadata, falling back to a curated set when offline.
func moduleDepOptions(md *generator.Metadata) []huh.Option[string] {
	if md != nil && len(md.Dependencies) > 0 {
		opts := make([]huh.Option[string], 0, len(md.Dependencies))
		for _, d := range md.Dependencies {
			opts = append(opts, huh.NewOption(d.Name, d.ID))
		}
		return opts
	}
	return []huh.Option[string]{
		huh.NewOption("Spring Web", "web"),
		huh.NewOption("Spring Security", "security"),
		huh.NewOption("Spring Data JPA", "data-jpa"),
		huh.NewOption("PostgreSQL Driver", "postgresql"),
		huh.NewOption("MySQL Driver", "mysql"),
		huh.NewOption("H2 Database", "h2"),
		huh.NewOption("Redis Cache", "data-redis"),
		huh.NewOption("Apache Kafka", "kafka"),
		huh.NewOption("Lombok", "lombok"),
	}
}

// CreateModuleWorkflow runs a fast, dedicated 3-step prompt (name, strategy,
// dependencies) instead of the full-screen TUI, then generates and applies the
// chosen strategy.
func CreateModuleWorkflow(md *generator.Metadata) {
	if md == nil {
		md = generator.FallbackMetadata()
	}

	// Prefer the latest GA over milestone/RC releases.
	bootVersion := generator.LatestGA(md)

	var moduleName, strategy string
	var deps []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Module Name").
				Placeholder("e.g., auth-service").
				Value(&moduleName),

			huh.NewSelect[string]().
				Title("Generation Strategy").
				Description("How the module is laid out on disk.").
				Options(
					huh.NewOption("Barebones (Stand-alone) - Full project with its own build wrappers", "barebones"),
					huh.NewOption("Strip Wrappers (Submodule) - Removes build wrappers to nest in a parent repo", "strip_wrappers"),
					huh.NewOption("Inherit from Parent - Like Submodule, but requires manual linking to parent POM/Gradle", "inherit"),
				).
				Value(&strategy),

			huh.NewMultiSelect[string]().
				Title("Dependencies").
				Description("Type to filter · space to toggle").
				Options(moduleDepOptions(md)...).
				Filterable(true).
				Height(12).
				Value(&deps),
		),
	).WithTheme(sproutTheme())

	if err := form.Run(); err != nil {
		log.Fatal(err)
	}

	if strings.TrimSpace(moduleName) == "" {
		fmt.Println("Operation cancelled.")
		return
	}

	cfg := generator.Config{
		BuildTool:   "maven-project",
		Language:    "java",
		BootVersion: bootVersion,
		GroupId:     "com.example",
		ArtifactId:  moduleName,
		Name:        moduleName,
		Description: "Module",
		PackageName: "com.example." + strings.ReplaceAll(moduleName, "-", ""),
		Packaging:   "jar",
		JavaVersion: "21",
		Deps:        deps,
	}

	fmt.Printf("\n🌱 Creating module '%s'...\n", moduleName)

	if err := generator.Generate(cfg); err != nil {
		log.Fatalf("Failed to generate module: %v", err)
	}

	dest := generator.FolderName(cfg)

	if strategy == "strip_wrappers" || strategy == "inherit" {
		wrappers := []string{"mvnw", "mvnw.cmd", "gradlew", "gradlew.bat", ".mvn", "gradle", ".gitignore", ".git"}
		for _, w := range wrappers {
			os.RemoveAll(filepath.Join(dest, w))
		}
		fmt.Println("✓ Stripped Maven/Gradle wrappers")
	}
	if strategy == "inherit" {
		fmt.Println("✓ (Inherit) Reminder: link this module to your parent POM/Gradle file.")
	}

	fmt.Printf("✓ Module created at ./%s\n", dest)
	// Submodules usually live inside an existing repo; only offer git for stand-alone.
	if strategy == "barebones" {
		maybeGitInit(dest)
	}
	printNextSteps(dest)
}
