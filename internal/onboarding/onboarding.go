// Package onboarding renders the first-run account setup: a credentials modal
// floating over a playful, self-typing background. It validates the account and
// performs the initial sync before handing control back to the main app.
package onboarding

import (
	"errors"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"thinx/internal/tui/uihelp"
)

// ErrInvalidCredentials should be returned by Deps.Verify when the username or
// password is rejected, so the modal can show a friendly message. Defined here so
// onboarding stays decoupled from the concrete backend adapter.
var ErrInvalidCredentials = errors.New("invalid username or password")

// Deps are the side effects the onboarding flow needs, injected by the caller.
type Deps struct {
	Verify    func(email, password string) error // check credentials, no persistence
	Save      func(email, password string) error // persist the validated account
	FirstSync func(email, password string) error // run the initial (slow) sync
}

const (
	boxMaxWidth = 76
	boxMargin   = 6
	labelWidth  = 11

	hintText = "New here? Buy Things 3 for iPhone, iPad & Mac from Cultured Code " +
		"and create your account there — then sign in below. You can also create a local, non-syncing database without an account."
)

// phase is the onboarding lifecycle state.
type phase int

const (
	phaseForm phase = iota // editing the credentials form
	phaseVerifying
	phaseSyncing
)

// focus targets within the form.
const (
	focusUsername = iota
	focusPassword
	focusCount // hack to get the number of focusable fields
)

type bgTickMsg struct{}
type verifiedMsg struct{ err error }
type syncedMsg struct{ err error }

type model struct {
	deps     Deps
	keys     keyMap
	bg       *background
	username textinput.Model
	password textinput.Model
	focus    int
	phase    phase
	errMsg   string
	width    int
	height   int
}

// Run shows the onboarding program and blocks until the user finishes or quits.
func Run(deps Deps) error {
	_, err := tea.NewProgram(newModel(deps)).Run()
	return err
}

func newModel(deps Deps) model {
	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "you@example.com"
	user.CharLimit = 254
	user.Focus()

	pass := textinput.New()
	pass.Prompt = ""
	pass.Placeholder = "password"
	pass.CharLimit = 254
	pass.EchoMode = textinput.EchoPassword

	return model{
		deps:     deps,
		keys:     newKeyMap(),
		bg:       newBackground(time.Now().UnixNano()),
		username: user,
		password: pass,
	}
}

func (m model) Init() tea.Cmd {
	tick := tea.Tick(typeInterval, func(time.Time) tea.Msg { return bgTickMsg{} })
	return tea.Batch(tick, m.username.Focus())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.bg.resize(msg.Width, msg.Height)
		m.layoutInputs()
		return m, nil
	case bgTickMsg:
		m.bg.advance()
		return m, tea.Tick(m.bg.next(), func(time.Time) tea.Msg { return bgTickMsg{} })
	case verifiedMsg:
		return m.onVerified(msg.err)
	case syncedMsg:
		if msg.err != nil {
			m.phase = phaseForm
			m.errMsg = "Initial sync failed: " + msg.err.Error()
			cmd := m.applyFocus()
			return m, cmd
		}
		return m, tea.Quit
	case tea.PasteMsg:
		if m.phase != phaseForm {
			return m, nil // ignore input while verifying or syncing
		}
		return m.updateFocused(msg)
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.quit) {
			return m, tea.Quit
		}
		if m.phase != phaseForm {
			return m, nil // ignore input while verifying or syncing
		}

		switch {
		case key.Matches(msg, m.keys.submit):
			return m.submit()
		case key.Matches(msg, m.keys.next):
			m.focus = (m.focus + 1) % focusCount
			cmd := m.applyFocus()
			return m, cmd
		case key.Matches(msg, m.keys.prev):
			m.focus = (m.focus + focusCount - 1) % focusCount
			cmd := m.applyFocus()
			return m, cmd
		default:
			return m.updateFocused(msg)
		}
	}
	return m, nil
}

// updateFocused forwards msg (a key press or paste) to whichever field is focused.
func (m model) updateFocused(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focus {
	case focusUsername:
		m.username, cmd = m.username.Update(msg)
	case focusPassword:
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}

// submit validates the form locally, then kicks off credential verification.
func (m model) submit() (tea.Model, tea.Cmd) {
	user := strings.TrimSpace(m.username.Value())
	pass := m.password.Value()
	if user == "" || pass == "" {
		m.errMsg = "Enter your username and password."
		return m, nil
	}
	m.errMsg = ""
	m.phase = phaseVerifying
	m.username.Blur()
	m.password.Blur()
	verify := m.deps.Verify
	return m, func() tea.Msg { return verifiedMsg{err: verify(user, pass)} }
}

// onVerified persists the account and starts the first sync once credentials check
// out; otherwise it returns to the form with an explanation.
func (m model) onVerified(err error) (tea.Model, tea.Cmd) {
	if err != nil {
		m.phase = phaseForm
		if errors.Is(err, ErrInvalidCredentials) {
			m.errMsg = "Incorrect username or password."
		} else {
			m.errMsg = "Could not verify: " + err.Error()
		}
		cmd := m.applyFocus()
		return m, cmd
	}

	user := strings.TrimSpace(m.username.Value())
	pass := m.password.Value()
	if err := m.deps.Save(user, pass); err != nil {
		m.phase = phaseForm
		m.errMsg = "Could not save config: " + err.Error()
		cmd := m.applyFocus()
		return m, cmd
	}

	m.phase = phaseSyncing
	sync := m.deps.FirstSync
	return m, func() tea.Msg { return syncedMsg{err: sync(user, pass)} }
}

// applyFocus moves keyboard focus to the field at m.focus.
func (m *model) applyFocus() tea.Cmd {
	m.username.Blur()
	m.password.Blur()
	switch m.focus {
	case focusUsername:
		return m.username.Focus()
	case focusPassword:
		return m.password.Focus()
	}
	return nil
}

func (m *model) layoutInputs() {
	w := m.contentWidth() - labelWidth - 1 // -1 for the cursor cell
	m.username.SetWidth(w)
	m.password.SetWidth(w)
}

// View renders the modal floating over the animated background.
func (m model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if m.width < 70 || m.height < 5 {
		v.Content = "Resize your terminal to set up thinx."
		return v
	}
	v.Content = overlay(m.bg.render(), m.modalBox(), m.width, m.height)
	return v
}

// overlay centers box over base using a lipgloss canvas, mirroring the task modal.
func overlay(base, box string, width, height int) string {
	if width <= 0 || height <= 0 {
		return box
	}
	x := (width - lipgloss.Width(box)) / 2
	y := (height - lipgloss.Height(box)) / 2
	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(base),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	))
	return canvas.Render()
}

func (m model) modalBox() string {
	rows := []string{
		titleStyle.Render("Set up thinx"),
		lipgloss.Wrap(hintText, m.contentWidth(), ""),
		"",
		m.row("Provider", providerStyle.Render("Things Cloud")),
		m.row("Username", m.username.View()),
		m.row("Password", m.password.View()),
		"",
		m.statusLine(),
		"",
		legendStyle.Render(m.legend()),
	}
	return boxStyle.Width(m.outerWidth()).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (m model) row(label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), value)
}

// statusLine shows the current progress or error. It always occupies one line so
// the modal height stays stable when verifying/syncing begins.
func (m model) statusLine() string {
	switch m.phase {
	case phaseVerifying:
		return statusStyle.Render("Verifying credentials…")
	case phaseSyncing:
		return statusStyle.Render("Initial sync, this might take a moment…")
	default:
		if m.errMsg != "" {
			return errorStyle.Render(m.errMsg)
		}
		return "" // reserve the line
	}
}

func (m model) legend() string {
	return strings.Join([]string{
		uihelp.BindingHelp(m.keys.next),
		uihelp.BindingHelp(m.keys.submit),
		uihelp.BindingHelp(m.keys.quit),
	}, "; ")
}

func (m model) outerWidth() int {
	return min(m.width-2*boxMargin, boxMaxWidth)
}

func (m model) contentWidth() int {
	return m.outerWidth() - boxStyle.GetHorizontalBorderSize() - boxStyle.GetHorizontalPadding()
}
