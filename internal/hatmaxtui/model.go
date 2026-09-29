package hatmaxtui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/interaction"
)

const (
	defaultWidth  = 80
	defaultHeight = 24
	composerLines = 4
)

// Session is the conversational kernel surface consumed by the TUI adapter.
type Session interface {
	Current() conversation.Conversation
	Turn(context.Context, conversation.TurnRequest) (conversation.SessionResult, error)
	Approve(context.Context, string, string) (conversation.SessionResult, error)
	Cancel(context.Context, string) (conversation.SessionResult, error)
	Reset(context.Context) (conversation.Conversation, error)
	Close() error
}

// SessionFactory opens one resumable conversation for a canonical root.
type SessionFactory func(
	context.Context,
	string,
	conversation.SessionOptions,
) (Session, error)

type keyMap struct {
	submit  key.Binding
	newline key.Binding
	approve key.Binding
	cancel  key.Binding
	reset   key.Binding
	help    key.Binding
	quit    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		submit:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		newline: key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("ctrl+j", "newline")),
		approve: key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "approve plan")),
		cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		reset:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new conversation")),
		help:    key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "more help")),
		quit:    key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	}
}

func (bindings keyMap) ShortHelp() []key.Binding {
	return []key.Binding{bindings.submit, bindings.approve, bindings.cancel, bindings.help, bindings.quit}
}

func (bindings keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{bindings.submit, bindings.newline, bindings.approve, bindings.cancel},
		{bindings.reset, bindings.help, bindings.quit},
	}
}

type sessionOpenedMsg struct {
	session Session
	value   conversation.Conversation
	err     error
}

type turnFinishedMsg struct {
	result conversation.SessionResult
	err    error
}

type resetFinishedMsg struct {
	value conversation.Conversation
	err   error
}

type model struct {
	ctx       context.Context
	root      string
	sessions  SessionFactory
	session   Session
	value     conversation.Conversation
	viewport  viewport.Model
	composer  textarea.Model
	help      help.Model
	progress  progress.Model
	keys      keyMap
	width     int
	height    int
	status    string
	detail    string
	pendingID string
	digest    string
	busy      bool
	cancel    context.CancelFunc
}

func newModel(ctx context.Context, root string, sessions SessionFactory) model {
	composer := textarea.New()
	composer.Placeholder = "Describe what you want to build with Hatmax"
	composer.Prompt = "> "
	composer.CharLimit = conversation.MaximumTurnBytes
	composer.KeyMap.InsertNewline.SetKeys("ctrl+j")
	composer.SetHeight(composerLines)
	composer.SetWidth(defaultWidth)
	composer.Focus()

	conversationView := viewport.New(
		viewport.WithWidth(defaultWidth),
		viewport.WithHeight(defaultHeight-composerLines-6),
	)
	conversationView.SetContent("Opening the local Hatmax conversation...")

	helpModel := help.New()
	helpModel.SetWidth(defaultWidth)

	progressModel := progress.New(progress.WithDefaultBlend())
	progressModel.SetWidth(defaultWidth)

	return model{
		ctx: ctx, root: root, sessions: sessions,
		viewport: conversationView,
		composer: composer,
		help:     helpModel,
		progress: progressModel,
		keys:     newKeyMap(),
		width:    defaultWidth, height: defaultHeight,
		status: "Opening conversation", busy: true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.openSession())
}

func (m model) openSession() tea.Cmd {
	return func() tea.Msg {
		if m.sessions == nil {
			return sessionOpenedMsg{err: fmt.Errorf("conversation session factory is unavailable")}
		}

		session, err := m.sessions(m.ctx, m.root, conversation.SessionOptions{})
		if err != nil {
			return sessionOpenedMsg{err: err}
		}

		return sessionOpenedMsg{session: session, value: session.Current()}
	}
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.resize(message.Width, message.Height)
	case sessionOpenedMsg:
		m.busy = false
		if message.err != nil {
			m.status = "Conversation unavailable"
			m.detail = message.err.Error()
			m.refreshViewport()

			return m, nil
		}

		m.session = message.session
		m.value = message.value
		m.status = "Ready"
		m.refreshViewport()

		return m, nil
	case turnFinishedMsg:
		m.finishWork(message.result, message.err)

		return m, nil
	case resetFinishedMsg:
		m.busy = false

		m.cancel = nil
		if message.err != nil {
			m.status = "Reset failed"
			m.detail = message.err.Error()
		} else {
			m.value = message.value
			m.pendingID = ""
			m.digest = ""
			m.detail = ""
			m.status = "New conversation"
		}

		m.refreshViewport()

		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(message, m.keys.quit):
			m.cancelWork()

			return m, tea.Quit
		case key.Matches(message, m.keys.help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(message, m.keys.approve):
			if command := m.approve(); command != nil {
				return m, command
			}
		case key.Matches(message, m.keys.reset):
			if command := m.reset(); command != nil {
				return m, command
			}
		case key.Matches(message, m.keys.cancel):
			if command := m.cancelOperation(); command != nil {
				return m, command
			}
		case key.Matches(message, m.keys.submit):
			if command := m.submit(); command != nil {
				return m, command
			}

			return m, nil
		}
	}

	var command tea.Cmd

	m.composer, command = m.composer.Update(message)
	commands = append(commands, command)
	m.viewport, command = m.viewport.Update(message)
	commands = append(commands, command)
	m.progress, command = m.progress.Update(message)
	commands = append(commands, command)

	return m, tea.Batch(commands...)
}

func (m *model) submit() tea.Cmd {
	content := strings.TrimSpace(m.composer.Value())
	if m.busy || m.session == nil || content == "" {
		return nil
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel = cancel
	m.busy = true
	m.status = "Interpreting"
	m.detail = ""
	m.composer.Reset()

	session := m.session

	return func() tea.Msg {
		result, err := session.Turn(ctx, conversation.TurnRequest{Content: content})

		return turnFinishedMsg{result: result, err: err}
	}
}

func (m *model) approve() tea.Cmd {
	if m.busy || m.session == nil || m.pendingID == "" || m.digest == "" {
		return nil
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel = cancel
	m.busy = true
	m.status = "Executing approved plan"

	session := m.session
	operationID := m.pendingID
	digest := m.digest

	return func() tea.Msg {
		result, err := session.Approve(ctx, operationID, digest)

		return turnFinishedMsg{result: result, err: err}
	}
}

func (m *model) cancelOperation() tea.Cmd {
	if m.busy {
		m.cancelWork()
		m.status = "Cancelling"

		return nil
	}

	if m.session == nil || m.pendingID == "" {
		m.composer.Reset()
		m.status = "Ready"

		return nil
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel = cancel
	m.busy = true
	m.status = "Cancelling proposal"

	session := m.session
	operationID := m.pendingID

	return func() tea.Msg {
		result, err := session.Cancel(ctx, operationID)

		return turnFinishedMsg{result: result, err: err}
	}
}

func (m *model) reset() tea.Cmd {
	if m.busy || m.session == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(m.ctx)
	m.cancel = cancel
	m.busy = true
	m.status = "Starting new conversation"

	session := m.session

	return func() tea.Msg {
		value, err := session.Reset(ctx)

		return resetFinishedMsg{value: value, err: err}
	}
}

func (m *model) finishWork(result conversation.SessionResult, err error) {
	m.busy = false
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}

	if err != nil {
		m.status = "Operation failed"
		m.detail = err.Error()
		m.refreshViewport()

		return
	}

	m.value = result.Conversation
	m.status = outcomeStatus(result.Interaction.Outcome)
	m.detail = resultDetail(result)
	m.pendingID = ""
	m.digest = ""

	if result.Interaction.Outcome == interaction.OutcomePlanReady && result.Interaction.Plan != nil {
		m.pendingID = result.OperationID
		m.digest = result.Interaction.Plan.Digest
	}

	m.refreshViewport()
}

func (m *model) cancelWork() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m model) close() {
	m.cancelWork()

	if m.session != nil {
		_ = m.session.Close()
	}
}

func (m *model) resize(width, height int) {
	if width < 1 || height < 1 {
		return
	}

	m.width = width
	m.height = height
	m.composer.SetWidth(width)
	m.help.SetWidth(width)
	m.progress.SetWidth(width)
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(max(1, height-composerLines-6))
}

func (m *model) refreshViewport() {
	m.viewport.SetContent(conversationPresentation(m.value, m.detail))
	m.viewport.GotoBottom()
}

func (m model) View() tea.View {
	heading := lipgloss.NewStyle().Bold(true).Render("Hatmax")
	status := lipgloss.NewStyle().Faint(true).Render("Status: " + m.status)

	parts := []string{heading, m.viewport.View(), m.composer.View(), status}
	if m.busy {
		parts = append(parts, m.progress.ViewAs(0.55))
	}

	parts = append(parts, m.help.View(m.keys))

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	view.AltScreen = true
	view.WindowTitle = "Hatmax"

	return view
}

func conversationPresentation(value conversation.Conversation, detail string) string {
	var output strings.Builder

	if len(value.Turns) == 0 {
		output.WriteString("Build and evolve Hatmax applications through conversation.")
	}

	for _, turn := range value.Turns {
		speaker := "You"
		if turn.Role == conversation.RoleHatmax {
			speaker = "Hatmax"
		}

		fmt.Fprintf(&output, "%s [%s]\n%s\n\n", speaker, turn.Kind, turn.Content)
	}

	if detail != "" {
		fmt.Fprintf(&output, "Details\n%s", detail)
	}

	return strings.TrimSpace(output.String())
}

func outcomeStatus(outcome interaction.Outcome) string {
	switch outcome {
	case interaction.OutcomeConversationResponse:
		return "Conversation"
	case interaction.OutcomeClarificationRequired:
		return "Clarification required"
	case interaction.OutcomePlanReady:
		return "Plan ready for approval"
	case interaction.OutcomeCompleted:
		return "Completed"
	case interaction.OutcomeValidationIncomplete:
		return "Completed; validation incomplete"
	case interaction.OutcomeUnsupported:
		return "Unsupported Hatmax request"
	case interaction.OutcomeIntentRejected:
		return "Intent rejected"
	case interaction.OutcomePlanStale:
		return "Plan stale"
	case interaction.OutcomeExecutionFailed:
		return "Execution failed"
	case interaction.OutcomeCancelled:
		return "Cancelled"
	default:
		return "Failed"
	}
}

func resultDetail(result conversation.SessionResult) string {
	var detail strings.Builder

	interactionResult := result.Interaction

	if len(interactionResult.PlanYAML) > 0 {
		fmt.Fprintf(&detail, "Plan\n%s", interactionResult.PlanYAML)

		if !strings.HasSuffix(detail.String(), "\n") {
			detail.WriteByte('\n')
		}
	}

	for _, diagnostic := range interactionResult.Diagnostics {
		fmt.Fprintf(&detail, "Diagnostic %s (%s): %s\n", diagnostic.Code, diagnostic.Phase, diagnostic.Message)
	}

	for _, change := range interactionResult.FreshnessChanges {
		fmt.Fprintf(&detail, "Stale %s: %s\n", change.Kind, change.Path)
	}

	for _, change := range interactionResult.RetainedChanges {
		fmt.Fprintf(&detail, "Retained %s: %s\n", change.Kind, change.Target)
	}

	if interactionResult.Report != nil {
		fmt.Fprintf(&detail, "Conformance: %t\n", interactionResult.Report.Conformance.Passed)

		for _, command := range interactionResult.Report.Commands {
			fmt.Fprintf(&detail, "Validation %s: exit %d\n", command.Name, command.ExitCode)
		}
	}

	if result.PersistenceFailed {
		detail.WriteString("Local conversation persistence failed; this session continues in memory only.\n")
	}

	return strings.TrimSpace(detail.String())
}
