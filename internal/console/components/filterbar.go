package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/console/styles"
)

type FilterBarModel struct {
	input textinput.Model
}

func NewFilterBarModel() FilterBarModel {
	ti := textinput.New()
	ti.Placeholder = "filter by name..."
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(styles.Muted).Italic(true)
	ti.Prompt = "▸ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(styles.Accent).Bold(true)
	return FilterBarModel{input: ti}
}

func (m FilterBarModel) Init() tea.Cmd { return nil }

func (m FilterBarModel) Update(msg tea.Msg) (FilterBarModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m FilterBarModel) View() string {
	return lipgloss.NewStyle().
		Background(styles.Surface).
		Foreground(styles.Primary).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(styles.Accent).
		BorderBackground(styles.Surface).
		Padding(0, 1).
		Render(m.input.View())
}

func (m FilterBarModel) Value() string {
	return m.input.Value()
}

func (m *FilterBarModel) Focus() tea.Cmd {
	return m.input.Focus()
}

func (m *FilterBarModel) Blur() {
	m.input.Blur()
	m.input.Reset()
}

func (m FilterBarModel) Focused() bool {
	return m.input.Focused()
}
