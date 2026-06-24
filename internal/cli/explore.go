package cli

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jessn-dev/sprout/internal/generator"
)

var exploreCmd = &cobra.Command{
	Use:   "explore",
	Short: "Explore the Spring ecosystem and learn about dependencies",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Fetching Spring Ecosystem metadata...")
		md, err := generator.FetchMetadata()
		if err != nil || md == nil {
			log.Fatal("Could not fetch Spring ecosystem metadata. Please check your internet connection.")
		}

		m := newExploreModel(md.Dependencies)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(exploreCmd)
}

type item struct {
	dep generator.Dependency
}

func (i item) Title() string       { return i.dep.Name }
func (i item) Description() string { return i.dep.Group + " • " + i.dep.ID }
func (i item) FilterValue() string { return i.dep.Name + " " + i.dep.Group + " " + i.dep.ID }

type exploreModel struct {
	list     list.Model
	width    int
	height   int
	quitting bool
}

func newExploreModel(deps []generator.Dependency) exploreModel {
	items := make([]list.Item, len(deps))
	for i, d := range deps {
		items[i] = item{dep: d}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("#22C55E")).BorderLeftForeground(lipgloss.Color("#22C55E"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#10B981")).BorderLeftForeground(lipgloss.Color("#22C55E"))

	m := list.New(items, delegate, 0, 0)
	m.Title = "Spring Ecosystem Explorer"
	m.Styles.Title = lipgloss.NewStyle().Background(lipgloss.Color("#A855F7")).Foreground(lipgloss.Color("#111827")).Padding(0, 1).Bold(true)

	return exploreModel{
		list: m,
	}
}

func (m exploreModel) Init() tea.Cmd {
	return nil
}

func (m exploreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't quit if the user is filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		leftW := m.width / 2
		if leftW < 40 {
			leftW = 40
		}
		if leftW > m.width {
			leftW = m.width
		}

		m.list.SetSize(leftW, m.height)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m exploreModel) View() string {
	if m.quitting {
		return ""
	}

	left := m.list.View()

	if m.width < 80 {
		return left // If terminal is too small, just show the list
	}

	var right string
	if i, ok := m.list.SelectedItem().(item); ok {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#22C55E")).MarginBottom(1)
		idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).MarginBottom(1)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Width(m.width/2 - 6)

		right = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(i.dep.Name),
			idStyle.Render(fmt.Sprintf("ID: %s | Group: %s", i.dep.ID, i.dep.Group)),
			descStyle.Render(i.dep.Description),
		)

		if i.dep.VersionRange != "" {
			right = lipgloss.JoinVertical(lipgloss.Left, right, "", lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("Requires Spring Boot: "+i.dep.VersionRange))
		}
	} else {
		right = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Select a dependency to view details...")
	}

	rightPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(m.width - m.list.Width() - 2).
		Height(m.height - 2).
		Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, rightPanel)
}
