// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package hatmaxtui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"hatmax.adrianpk.com/generator/conversation"
	"hatmax.adrianpk.com/generator/interaction"
)

const (
	defaultWidth     = 80
	defaultHeight    = 24
	composerMaxLines = 6
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
	details key.Binding
	quit    key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		submit:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "send")),
		newline: key.NewBinding(key.WithKeys("ctrl+j"), key.WithHelp("ctrl+j", "newline")),
		approve: key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "approve plan")),
		cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		reset:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new conversation")),
		help:    key.NewBinding(key.WithKeys("f1", "ctrl+h"), key.WithHelp("f1", "help")),
		details: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "details")),
		quit:    key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
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

type activityTickMsg struct {
	id uint64
}

type model struct {
	ctx           context.Context
	root          string
	sessions      SessionFactory
	session       Session
	value         conversation.Conversation
	viewport      viewport.Model
	composer      textarea.Model
	keys          keyMap
	width         int
	height        int
	status        string
	summary       string
	detail        string
	showDetail    bool
	showHelp      bool
	pendingID     string
	digest        string
	busy          bool
	activityID    uint64
	activityFrame int
	cancel        context.CancelFunc
}

func newModel(ctx context.Context, root string, sessions SessionFactory) model {
	composer := textarea.New()
	composer.Placeholder = "Describe what you want to build with Hatmax"
	composer.ShowLineNumbers = false
	composer.SetPromptFunc(composerPromptWidth, composerPrompt)
	composer.CharLimit = conversation.MaximumTurnBytes
	composer.KeyMap.InsertNewline.SetKeys("ctrl+j")
	composer.MaxHeight = composerMaxLines
	composer.SetHeight(1)
	composer.SetWidth(defaultWidth - composerHorizontalFrame)
	composer.SetStyles(composerStyles(composer.Styles()))
	composer.Focus()

	conversationView := viewport.New(
		viewport.WithWidth(defaultWidth),
		viewport.WithHeight(defaultHeight-4),
	)
	conversationView.SetContent("Opening the local Hatmax conversation...")

	value := model{
		ctx: ctx, root: root, sessions: sessions,
		viewport: conversationView,
		composer: composer,
		keys:     newKeyMap(),
		width:    defaultWidth, height: defaultHeight,
		status: "Opening", busy: true, activityID: 1,
	}
	value.resizeSurfaces()

	return value
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.openSession(), activityTick(m.activityID))
}

func activityTick(id uint64) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return activityTickMsg{id: id}
	})
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
	case activityTickMsg:
		if !m.busy || message.id != m.activityID {
			return m, nil
		}

		m.activityFrame++

		return m, activityTick(m.activityID)
	case sessionOpenedMsg:
		m.busy = false
		if message.err != nil {
			m.status = "Unavailable"
			m.summary = failureSummary("Hatmax could not open the local conversation.", message.err)
			m.detail = message.err.Error()
			m.resizeSurfaces()
			m.refreshViewport()

			return m, nil
		}

		m.session = message.session
		m.value = message.value
		m.status = "Ready"
		m.resizeSurfaces()
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
			m.summary = failureSummary("Hatmax could not start a new conversation.", message.err)
			m.detail = message.err.Error()
		} else {
			m.value = message.value
			m.pendingID = ""
			m.digest = ""
			m.summary = ""
			m.detail = ""
			m.showDetail = false
			m.status = "New conversation"
		}

		m.resizeSurfaces()
		m.refreshViewport()

		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(message, m.keys.quit):
			m.cancelWork()

			return m, tea.Quit
		case key.Matches(message, m.keys.help):
			m.showHelp = !m.showHelp

			return m, nil
		case key.Matches(message, m.keys.details) && m.detail != "":
			m.showDetail = !m.showDetail
			m.refreshViewport()

			return m, nil
		case key.Matches(message, m.keys.approve):
			if command := m.approve(); command != nil {
				return m, tea.Batch(command, activityTick(m.activityID))
			}
		case key.Matches(message, m.keys.reset):
			if command := m.reset(); command != nil {
				return m, tea.Batch(command, activityTick(m.activityID))
			}
		case key.Matches(message, m.keys.cancel):
			if command := m.cancelOperation(); command != nil {
				return m, tea.Batch(command, activityTick(m.activityID))
			}
		case key.Matches(message, m.keys.submit):
			if command := m.submit(); command != nil {
				return m, tea.Batch(command, activityTick(m.activityID))
			}

			return m, nil
		}
	}

	var command tea.Cmd

	m.composer, command = m.composer.Update(message)
	commands = append(commands, command)

	m.resizeComposer()
	m.resizeSurfaces()
	m.viewport, command = m.viewport.Update(message)
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
	m.beginActivity("Planning")
	m.summary = ""
	m.detail = ""
	m.showDetail = false
	m.composer.Reset()
	m.resizeComposer()
	m.resizeSurfaces()

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
	m.beginActivity("Executing")

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
	m.beginActivity("Cancelling")

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
	m.beginActivity("Resetting")

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
		m.status = "Failed"
		m.summary = failureSummary("Hatmax could not complete the operation.", err)
		m.detail = err.Error()
		m.showDetail = false
		m.resizeSurfaces()
		m.refreshViewport()

		return
	}

	m.value = result.Conversation
	m.status = outcomeStatus(result.Interaction.Outcome)
	m.summary = resultSummary(result)
	m.detail = resultDetail(result)
	m.showDetail = false
	m.pendingID = ""
	m.digest = ""

	if result.Interaction.Outcome == interaction.OutcomePlanReady && result.Interaction.Plan != nil {
		m.pendingID = result.OperationID
		m.digest = result.Interaction.Plan.Digest
	}

	m.resizeSurfaces()
	m.refreshViewport()
}

func (m *model) cancelWork() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *model) beginActivity(status string) {
	m.busy = true
	m.status = status
	m.activityID++
	m.activityFrame = 0
	m.resizeSurfaces()
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
	m.composer.SetWidth(max(1, width-composerHorizontalFrame))
	m.resizeComposer()
	m.resizeSurfaces()
	m.refreshViewport()
}

func (m *model) refreshViewport() {
	m.viewport.SetContent(conversationPresentation(
		m.value,
		m.summary,
		m.detail,
		m.showDetail,
		m.viewport.Width(),
	))
	m.viewport.GotoBottom()
}

func (m model) View() tea.View {
	parts := []string{headingView()}
	if m.busy {
		parts = append(parts, activityView(m.width, m.activityFrame))
	}

	parts = append(parts, m.viewport.View(), composerView(m.composer.View(), m.width), m.footerView())

	view := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, parts...))
	view.AltScreen = true
	view.WindowTitle = "Hatmax"

	return view
}
