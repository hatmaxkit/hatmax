package hatmaxtui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
	composerLines = 4
)

type keyMap struct {
	submit key.Binding
	clear  key.Binding
	help   key.Binding
	quit   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		clear:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear/cancel")),
		help:   key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "more help")),
		quit:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
}

func (bindings keyMap) ShortHelp() []key.Binding {
	return []key.Binding{bindings.submit, bindings.clear, bindings.help, bindings.quit}
}

func (bindings keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{bindings.submit, bindings.clear}, {bindings.help, bindings.quit}}
}

type model struct {
	viewport viewport.Model
	composer textarea.Model
	help     help.Model
	keys     keyMap
	width    int
	height   int
	status   string
}

func newModel() model {
	composer := textarea.New()
	composer.Placeholder = "Describe what you want to build with Hatmax"
	composer.Prompt = "> "
	composer.CharLimit = 32 << 10
	composer.SetHeight(composerLines)
	composer.SetWidth(defaultWidth)
	composer.Focus()

	conversation := viewport.New(
		viewport.WithWidth(defaultWidth),
		viewport.WithHeight(defaultHeight-composerLines-5),
	)
	conversation.SetContent("Hatmax\n\nBuild and evolve Hatmax applications through conversation.")

	helpModel := help.New()
	helpModel.SetWidth(defaultWidth)

	return model{
		viewport: conversation,
		composer: composer,
		help:     helpModel,
		keys:     newKeyMap(),
		width:    defaultWidth,
		height:   defaultHeight,
		status:   "Ready",
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.resize(message.Width, message.Height)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(message, m.keys.quit):
			return m, tea.Quit
		case key.Matches(message, m.keys.help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(message, m.keys.clear):
			m.composer.Reset()
			m.status = "Cancelled"
		case key.Matches(message, m.keys.submit):
			if strings.TrimSpace(m.composer.Value()) != "" {
				m.status = "Preparing Hatmax turn"
			}

			return m, nil
		}
	}

	var command tea.Cmd
	m.composer, command = m.composer.Update(message)
	commands = append(commands, command)
	m.viewport, command = m.viewport.Update(message)
	commands = append(commands, command)

	return m, tea.Batch(commands...)
}

func (m *model) resize(width, height int) {
	if width < 1 || height < 1 {
		return
	}

	m.width = width
	m.height = height
	m.composer.SetWidth(width)
	m.help.SetWidth(width)
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(max(1, height-composerLines-5))
}

func (m model) View() tea.View {
	heading := lipgloss.NewStyle().Bold(true).Render("Hatmax")
	status := lipgloss.NewStyle().Faint(true).Render("Status: " + m.status)
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		heading,
		m.viewport.View(),
		m.composer.View(),
		status,
		m.help.View(m.keys),
	)

	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "Hatmax"

	return view
}
