package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jessn-dev/sprout/internal/generator"
	"github.com/jessn-dev/sprout/internal/version"
)

type option struct {
	value string
	label string
}

var (
	buildTools = []option{
		{"maven-project", "Maven"},
		{"gradle-project", "Gradle - Groovy"},
		{"gradle-project-kotlin", "Gradle - Kotlin"},
	}
	languages = []option{
		{"java", "Java"},
		{"kotlin", "Kotlin"},
		{"groovy", "Groovy"},
	}
	packagings = []option{
		{"jar", "JAR"},
		{"war", "WAR"},
	}
	javaVersions = []option{
		{"17", "17"},
		{"21", "21"},
		{"23", "23"},
		{"24", "24"},
		{"25", "25"},
		{"26", "26"},
	}
	projectTypes = []option{
		{"standard", "Spring Boot (Standard)"},
		{"cloud", "Spring Cloud (Microservices)"},
		{"security", "Spring Security (Secured Web App)"},
	}
)

// Focusable items, in tab order.
const (
	fBuild = iota
	fLang
	fGroup
	fArtifact
	fName
	fDesc
	fPkg
	fPack
	fJava
	fBoot
	fType
	fDeps // the dependency filter+list widget
	fGen
	numFields
)

const depListVisible = 12 // rows of dependencies shown at once

type Model struct {
	width, height int

	Config   generator.Config
	Generate bool

	bootVersions []generator.BootVersion
	javaVers     []option
	allDeps      []generator.Dependency

	focus int

	selBuild, selLang, selPack, selJava, selBoot, selType int

	inputs []textinput.Model // group, artifact, name, desc, pkg

	depFilter   textinput.Model
	depCursor   int
	depSelected map[string]bool

	genError        string // validation message shown near the Generate button
	confirmIncompat bool   // armed when incompatible deps need a 2nd Enter

	quitting bool
}

func NewModel(md *generator.Metadata) *Model {
	m := &Model{
		selJava:     1, // 21
		depSelected: map[string]bool{},
		javaVers:    javaVersions, // fallback list
	}

	if md != nil {
		m.bootVersions = md.BootVersions
		m.allDeps = md.Dependencies
		// Default to the latest GA release (avoid milestone/RC defaults).
		ga := generator.LatestGA(md)
		for i, v := range m.bootVersions {
			if v.ID == ga {
				m.selBoot = i
				break
			}
		}
		// Java versions from live metadata, defaulting to the server default.
		if len(md.JavaVersions) > 0 {
			m.javaVers = nil
			for i, jv := range md.JavaVersions {
				m.javaVers = append(m.javaVers, option{jv, jv})
				if jv == md.JavaDefault {
					m.selJava = i
				}
			}
		}
	}
	if len(m.bootVersions) == 0 {
		// Empty ID → omitted from request → server picks its default.
		m.bootVersions = []generator.BootVersion{{ID: "", Name: "Server Default"}}
	}

	labels := []string{"com.example", "demo", "demo", "Demo project for Spring Boot", "com.example.demo"}
	for _, def := range labels {
		ti := textinput.New()
		ti.SetValue(def)
		ti.Prompt = ""
		ti.CharLimit = 80
		ti.Width = 36
		m.inputs = append(m.inputs, ti)
	}

	df := textinput.New()
	df.Prompt = "🔍 "
	df.Placeholder = "type to filter..."
	df.CharLimit = 40
	df.Width = 28
	m.depFilter = df

	m.syncFocus()
	return m
}

func (m *Model) Init() tea.Cmd { return textinput.Blink }

func (m *Model) bootVersionID() string {
	if m.selBoot < len(m.bootVersions) {
		return m.bootVersions[m.selBoot].ID
	}
	return ""
}

func (m *Model) syncFocus() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	m.depFilter.Blur()
	switch m.focus {
	case fGroup, fArtifact, fName, fDesc, fPkg:
		m.inputs[m.focus-fGroup].Focus()
	case fDeps:
		m.depFilter.Focus()
	}
}

func (m *Model) focusNext() { m.clearGenState(); m.focus = (m.focus + 1) % numFields; m.syncFocus() }
func (m *Model) focusPrev() {
	m.clearGenState()
	m.focus = (m.focus - 1 + numFields) % numFields
	m.syncFocus()
}

// clearGenState resets validation/confirm prompts when the user moves on.
func (m *Model) clearGenState() { m.genError = ""; m.confirmIncompat = false }

func (m *Model) isInput() bool {
	switch m.focus {
	case fGroup, fArtifact, fName, fDesc, fPkg:
		return true
	}
	return false
}

func cycle(idx, n, dir int) int {
	if n == 0 {
		return 0
	}
	return (idx + dir + n) % n
}

// filteredDeps returns deps matching the current filter text.
func (m *Model) filteredDeps() []generator.Dependency {
	q := strings.ToLower(strings.TrimSpace(m.depFilter.Value()))
	if q == "" {
		return m.allDeps
	}
	var out []generator.Dependency
	for _, d := range m.allDeps {
		if strings.Contains(strings.ToLower(d.Name), q) ||
			strings.Contains(strings.ToLower(d.ID), q) ||
			strings.Contains(strings.ToLower(d.Group), q) {
			out = append(out, d)
		}
	}
	return out
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.focusNext()
			return m, nil
		case "shift+tab":
			m.focusPrev()
			return m, nil
		}

		// Dependency widget: type to filter, up/down to move, space to toggle.
		if m.focus == fDeps {
			return m.updateDeps(key, msg)
		}

		// Up/Down navigate between fields outside the dep widget.
		if key == "up" {
			m.focusPrev()
			return m, nil
		}
		if key == "down" {
			m.focusNext()
			return m, nil
		}

		if m.focus == fGen {
			if key == "enter" || key == " " {
				m.commit()
				if err := generator.ValidateConfig(m.Config); err != nil {
					m.genError = err.Error()
					m.confirmIncompat = false
					return m, nil
				}
				m.genError = ""
				if len(m.incompatibleDeps()) > 0 && !m.confirmIncompat {
					m.confirmIncompat = true // require a second Enter to proceed
					return m, nil
				}
				m.Generate = true
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.isInput() {
			if key == "enter" {
				m.focusNext()
				return m, nil
			}
			var cmd tea.Cmd
			i := m.focus - fGroup
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			return m, cmd
		}

		switch m.focus {
		case fBuild:
			m.cycleKey(key, &m.selBuild, len(buildTools))
		case fLang:
			m.cycleKey(key, &m.selLang, len(languages))
		case fPack:
			m.cycleKey(key, &m.selPack, len(packagings))
		case fJava:
			m.cycleKey(key, &m.selJava, len(m.javaVers))
		case fBoot:
			m.cycleKey(key, &m.selBoot, len(m.bootVersions))
		case fType:
			m.cycleKey(key, &m.selType, len(projectTypes))
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateDeps(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.filteredDeps()
	switch key {
	case "down":
		if m.depCursor < len(list)-1 {
			m.depCursor++
		}
		return m, nil
	case "up":
		if m.depCursor > 0 {
			m.depCursor--
		}
		return m, nil
	case " ", "enter":
		if m.depCursor < len(list) {
			id := list[m.depCursor].ID
			m.depSelected[id] = !m.depSelected[id]
		}
		return m, nil
	}
	// Anything else edits the filter; reset cursor into range.
	var cmd tea.Cmd
	m.depFilter, cmd = m.depFilter.Update(msg)
	if m.depCursor >= len(m.filteredDeps()) {
		m.depCursor = 0
	}
	return m, cmd
}

func (m *Model) cycleKey(key string, sel *int, n int) {
	switch key {
	case "left":
		*sel = cycle(*sel, n, -1)
	case "right", " ", "enter":
		*sel = cycle(*sel, n, +1)
	}
}

// incompatibleDeps lists selected deps whose version range excludes the
// chosen Boot version.
func (m *Model) incompatibleDeps() []generator.Dependency {
	boot := m.bootVersionID()
	var out []generator.Dependency
	for _, d := range m.allDeps {
		if m.depSelected[d.ID] && !generator.VersionCompatible(boot, d.VersionRange) {
			out = append(out, d)
		}
	}
	return out
}

func (m *Model) commit() {
	m.Config.BuildTool = buildTools[m.selBuild].value
	m.Config.Language = languages[m.selLang].value
	m.Config.Packaging = packagings[m.selPack].value
	m.Config.JavaVersion = m.javaVers[m.selJava].value
	m.Config.BootVersion = m.bootVersionID()
	m.Config.ProjectType = projectTypes[m.selType].value
	m.Config.GroupId = m.inputs[0].Value()
	m.Config.ArtifactId = m.inputs[1].Value()
	m.Config.Name = m.inputs[2].Value()
	m.Config.Description = m.inputs[3].Value()
	m.Config.PackageName = m.inputs[4].Value()

	m.Config.Deps = m.Config.Deps[:0]
	for _, d := range m.allDeps {
		if m.depSelected[d.ID] {
			m.Config.Deps = append(m.Config.Deps, d.ID)
		}
	}
}

// ---------- view ----------

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "Loading..."
	}

	// Natural column widths with a small gap; avoids a big empty gutter on
	// wide terminals.
	leftW := 58
	rightW := 52
	if leftW > m.width/2 {
		leftW = m.width / 2
	}
	if rightW > m.width-leftW-4 {
		rightW = m.width - leftW - 4
	}
	if rightW < 34 {
		rightW = 34
	}

	left := lipgloss.JoinVertical(lipgloss.Left,
		m.renderSettings(),
		"",
		m.renderMetadata(),
	)
	leftPanel := lipgloss.NewStyle().Width(leftW).Render(left)
	rightPanel := PanelStyle.Width(rightW).Render(m.renderDependencies())

	cols := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "    ", rightPanel)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		"",
		cols,
		m.renderWarnings(),
		"",
		m.renderGenerate(),
		m.renderFooter(),
	)
}

var bannerLines = []string{
	"  ███████╗ ██████╗  ██████╗   ██████╗  ██╗   ██╗ ████████╗",
	"  ██╔════╝ ██╔══██╗ ██╔══██╗ ██╔═══██╗ ██║   ██║ ╚══██╔══╝",
	"  ███████╗ ██████╔╝ ██████╔╝ ██║   ██║ ██║   ██║    ██║   ",
	"  ╚════██║ ██╔═══╝  ██╔══██╗ ██║   ██║ ██║   ██║    ██║   ",
	"  ███████║ ██║      ██║  ██║ ╚██████╔╝ ╚██████╔╝    ██║   ",
	"  ╚══════╝ ╚═╝      ╚═╝  ╚═╝  ╚═════╝   ╚═════╝     ╚═╝   ",
}

var bannerColors = []lipgloss.Color{"#22C55E", "#10B981", "#14B8A6", "#06B6D4", "#0EA5E9", "#3B82F6"}

func (m *Model) renderHeader() string {
	ver := lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent).Bold(true).Padding(0, 1).Render(version.String())

	// Tight terminals: compact 2-line logo instead of the full block.
	if m.height > 0 && m.height < 30 {
		var letters strings.Builder
		letters.WriteString("🌱 ")
		for i, r := range "SPROUT" {
			c := bannerColors[i%len(bannerColors)]
			letters.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(r)))
		}
		title := letters.String()
		tag := SubtitleStyle.Italic(true).Render("  The Ultimate Custom Spring Initializer CLI")
		return title + tag + "  " + ver
	}

	var b strings.Builder
	for i, ln := range bannerLines {
		b.WriteString(lipgloss.NewStyle().Foreground(bannerColors[i]).Bold(true).Render(ln) + "\n")
	}
	tag := SubtitleStyle.Italic(true).Render("  🌱 The Ultimate Custom Spring Initializer CLI")
	b.WriteString(tag + "  " + ver)
	return b.String()
}

func section(num, title string) string {
	return SectionNumStyle.Render(num) + " " + SectionTitleStyle.Render(title)
}

func (m *Model) renderSettings() string {
	var b strings.Builder
	b.WriteString(section("1", "PROJECT SETTINGS") + "\n")
	b.WriteString(MutedStyle.Render("Configure base project type and language.") + "\n\n")
	build := m.card("Build Tool", buildTools[m.selBuild].label, m.focus == fBuild)
	lang := m.card("Language", languages[m.selLang].label, m.focus == fLang)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, build, "  ", lang))
	return b.String()
}

func (m *Model) card(title, value string, focused bool) string {
	style := CardStyle
	mark := "  "
	if focused {
		style = CardSelectedStyle
		mark = "◀ "
	}
	body := lipgloss.NewStyle().Bold(true).Render(title) + "\n" + mark + value
	if focused {
		body += " ▶"
	}
	return style.Width(26).Render(body)
}

func (m *Model) renderMetadata() string {
	var b strings.Builder
	b.WriteString(section("2", "PROJECT METADATA") + "\n")
	b.WriteString(MutedStyle.Render("Define artifact coordinates and packaging.") + "\n\n")

	rows := []struct {
		label string
		idx   int
	}{
		{"Group ID", fGroup},
		{"Artifact ID", fArtifact},
		{"Project Name", fName},
		{"Description", fDesc},
		{"Package Name", fPkg},
	}
	for _, r := range rows {
		b.WriteString(m.inputRow(r.label, r.idx) + "\n")
	}
	b.WriteString(m.selectRow("Packaging", packagings[m.selPack].label, m.focus == fPack) + "\n")
	b.WriteString(m.selectRow("Java Version", m.javaVers[m.selJava].label, m.focus == fJava) + "\n")
	b.WriteString(m.selectRow("Boot Version", m.bootVersions[m.selBoot].Name, m.focus == fBoot) + "\n")
	return b.String()
}

func (m *Model) inputRow(label string, idx int) string {
	focused := m.focus == idx
	lbl := LabelStyle.Render(label)
	if focused {
		lbl = LabelFocusStyle.Render(label)
	}
	field := m.inputs[idx-fGroup].View()
	if !focused {
		field = ValueStyle.Render(field)
	}
	return fmt.Sprintf("%-16s %s", lbl, field)
}

func (m *Model) selectRow(label, value string, focused bool) string {
	lbl := LabelStyle.Render(label)
	if focused {
		lbl = LabelFocusStyle.Render(label)
	}
	var val string
	if focused {
		val = lipgloss.NewStyle().Foreground(ColorPrimary).Render("◀ " + value + " ▶")
	} else {
		val = ValueStyle.Render(value)
	}
	return fmt.Sprintf("%-16s %s", lbl, val)
}

func (m *Model) renderDependencies() string {
	var b strings.Builder
	b.WriteString(section("3", "DEPENDENCIES") + "\n")
	b.WriteString(MutedStyle.Render("Select modules to auto-configure.") + "\n\n")
	b.WriteString(m.selectRow("Profile", projectTypes[m.selType].label, m.focus == fType) + "\n\n")

	active := m.focus == fDeps
	b.WriteString(m.depFilter.View() + "\n\n")

	vis := m.depVisibleRows()
	list := m.filteredDeps()
	if len(list) == 0 {
		b.WriteString(MutedStyle.Render("  no matches") + "\n")
	}

	// Scroll window around the cursor.
	start := 0
	if m.depCursor >= vis {
		start = m.depCursor - vis + 1
	}
	end := start + vis
	if end > len(list) {
		end = len(list)
	}

	boot := m.bootVersionID()
	for i := start; i < end; i++ {
		d := list[i]
		box := "[ ]"
		if m.depSelected[d.ID] {
			box = lipgloss.NewStyle().Foreground(ColorPrimary).Render("[✓]")
		}
		name := d.Name
		if !generator.VersionCompatible(boot, d.VersionRange) {
			name += " " + lipgloss.NewStyle().Foreground(ColorWarn).Render("⚠")
		}
		cursor := "  "
		if active && i == m.depCursor {
			cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Render("› ")
			name = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(d.Name)
			if !generator.VersionCompatible(boot, d.VersionRange) {
				name += " " + lipgloss.NewStyle().Foreground(ColorWarn).Render("⚠")
			}
		}
		b.WriteString(cursor + box + " " + name + "\n")
	}

	if len(list) > vis {
		b.WriteString(MutedStyle.Render(fmt.Sprintf("\n  %d of %d shown", end-start, len(list))))
	}

	// Description of the highlighted dependency.
	if active && m.depCursor < len(list) {
		if desc := strings.TrimSpace(list[m.depCursor].Description); desc != "" {
			b.WriteString("\n" + MutedStyle.Italic(true).Render(wrap(desc, 44)))
		}
	}
	return b.String()
}

// wrap soft-wraps text to width on word boundaries.
func wrap(s string, width int) string {
	words := strings.Fields(s)
	var lines []string
	line := ""
	for _, w := range words {
		if line == "" {
			line = w
		} else if len(line)+1+len(w) <= width {
			line += " " + w
		} else {
			lines = append(lines, line)
			line = w
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// depVisibleRows fits the dependency list to the terminal height so the
// banner and surrounding chrome stay on screen.
func (m *Model) depVisibleRows() int {
	rows := m.height - 22 // banner + panel chrome + generate/footer overhead
	if rows < 4 {
		rows = 4
	}
	if rows > depListVisible {
		rows = depListVisible
	}
	return rows
}

func (m *Model) renderWarnings() string {
	bad := m.incompatibleDeps()
	if len(bad) == 0 {
		return ""
	}
	boot := m.bootVersionID()
	var lines []string
	header := WarnStyle.Render(fmt.Sprintf("⚠  %d dependency(s) incompatible with Boot %s:", len(bad), boot))
	lines = append(lines, header)
	for _, d := range bad {
		lines = append(lines, MutedStyle.Render(fmt.Sprintf("   • %s requires %s", d.Name, d.VersionRange)))
	}
	return "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderGenerate() string {
	style := GenerateStyle
	if m.focus == fGen {
		style = GenerateFocusStyle
	}
	btn := style.Render("🚀 GENERATE PROJECT")
	hint := MutedStyle.Render("  press Enter to sprout")
	out := btn + hint

	if m.genError != "" {
		out += "\n" + WarnStyle.Render("✗ "+strings.ReplaceAll(m.genError, "\n", " "))
	} else if m.confirmIncompat {
		out += "\n" + WarnStyle.Render("⚠ Incompatible deps selected — press Enter again to generate anyway, or Tab to fix.")
	}
	return out
}

func (m *Model) renderFooter() string {
	return "\n" + MutedStyle.Render("Tab move · ↑↓ select/scroll · ←→/space change · type to filter deps · Enter generate · Esc quit")
}
