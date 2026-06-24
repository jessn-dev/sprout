package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#22C55E") // green
	ColorSecondary = lipgloss.Color("#3B82F6") // blue
	ColorAccent    = lipgloss.Color("#A855F7") // purple
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorText      = lipgloss.Color("#E5E7EB")
	ColorBg        = lipgloss.Color("#111827")
	ColorWarn      = lipgloss.Color("#F59E0B") // amber

	WarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Numbered section header, e.g. ( 1 ) PROJECT SETTINGS
	SectionNumStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorAccent).
			Bold(true).
			Padding(0, 1)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	LabelStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	LabelFocusStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	ValueStyle = lipgloss.NewStyle().Foreground(ColorSecondary)

	MutedStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	CardSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Foreground(ColorPrimary).
				Padding(0, 1)

	GenerateStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorPrimary).
			Bold(true).
			Padding(0, 3)

	GenerateFocusStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorSecondary).
				Bold(true).
				Padding(0, 3)
)
