package cli

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jessn-dev/sprout/internal/generator"
	"github.com/jessn-dev/sprout/internal/tui"
	"github.com/jessn-dev/sprout/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "sprout",
	Short:   "Sprout is a custom Spring Boot initializer CLI",
	Long:    `A fast and interactive CLI tool to generate Spring Boot projects with opinionated setups.`,
	Version: version.String(),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg generator.Config
		var action string

		fmt.Print(getBanner())

		md, err := generator.FetchMetadata()
		if err != nil {
			fmt.Println("⚠️  Could not reach start.spring.io, using offline defaults.")
			md = generator.FallbackMetadata()
		}

		err = huh.NewSelect[string]().
			Title("What would you like to do?").
			Options(
				huh.NewOption("Initialize a new Spring App", "new_app"),
				huh.NewOption("Create a module immediately", "new_module"),
			).
			Value(&action).
			Run()

		if err != nil {
			log.Fatal(err)
		}

		if action == "new_module" {
			CreateModuleWorkflow(md)
			return
		}

		m := tui.NewModel(md)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}

		if !m.Generate {
			return // They quit early
		}

		cfg = m.Config

		// Defaults
		if cfg.GroupId == "" {
			cfg.GroupId = "com.example"
		}
		if cfg.ArtifactId == "" {
			cfg.ArtifactId = "demo"
		}
		if cfg.Name == "" {
			cfg.Name = "demo"
		}
		if cfg.Description == "" {
			cfg.Description = "Demo project for Spring Boot"
		}
		if cfg.PackageName == "" {
			cfg.PackageName = cfg.GroupId + "." + strings.ReplaceAll(cfg.ArtifactId, "-", "")
		}

		fmt.Printf("\n🌱 Sprouting your %s project on %s...\n", cfg.Name, cfg.Language)
		fmt.Printf("📦 Selected Modules: %v\n", cfg.Deps)

		if err := generator.Generate(cfg); err != nil {
			log.Fatalf("Failed to generate project: %v", err)
		}

		dir := generator.FolderName(cfg)
		maybeGitInit(dir)
		printNextSteps(dir)
	},
}

// maybeGitInit asks (interactively) whether to initialize a git repo.
func maybeGitInit(dir string) {
	var yes bool
	if err := huh.NewConfirm().Title("Initialize a git repository?").Value(&yes).WithTheme(sproutTheme()).Run(); err != nil {
		return
	}
	if yes {
		gitInit(dir)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func getBanner() string {
	lines := []string{
		"  ███████╗ ██████╗  ██████╗   ██████╗  ██╗   ██╗ ████████╗",
		"  ██╔════╝ ██╔══██╗ ██╔══██╗ ██╔═══██╗ ██║   ██║ ╚══██╔══╝",
		"  ███████╗ ██████╔╝ ██████╔╝ ██║   ██║ ██║   ██║    ██║   ",
		"  ╚════██║ ██╔═══╝  ██╔══██╗ ██║   ██║ ██║   ██║    ██║   ",
		"  ███████║ ██║      ██║  ██║ ╚██████╔╝ ╚██████╔╝    ██║   ",
		"  ╚══════╝ ╚═╝      ╚═╝  ╚═╝  ╚═════╝   ╚═════╝     ╚═╝   ",
	}
	// Green → blue vertical gradient.
	colors := []string{"#22C55E", "#10B981", "#14B8A6", "#06B6D4", "#0EA5E9", "#3B82F6"}

	var b strings.Builder
	b.WriteString("\n")
	for i, ln := range lines {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i])).Bold(true).Render(ln) + "\n")
	}

	tag := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true).
		Render("  🌱 The Ultimate Custom Spring Initializer CLI")
	ver := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111827")).
		Background(lipgloss.Color("#A855F7")).
		Bold(true).Padding(0, 1).Render(version.String())
	b.WriteString("\n" + tag + "  " + ver + "\n\n")
	return b.String()
}
