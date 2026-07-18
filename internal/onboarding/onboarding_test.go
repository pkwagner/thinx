package onboarding

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testDeps(verifyErr, saveErr, syncErr error, saved, synced *bool) Deps {
	return Deps{
		Verify: func(string, string) error { return verifyErr },
		Save: func(string, string) error {
			if saved != nil {
				*saved = true
			}
			return saveErr
		},
		FirstSync: func(string, string) error {
			if synced != nil {
				*synced = true
			}
			return syncErr
		},
	}
}

// TestOnboardingHappyPath walks submit -> verify -> save -> sync -> quit.
func TestOnboardingHappyPath(t *testing.T) {
	t.Parallel()
	var saved, synced bool
	m := newModel(testDeps(nil, nil, nil, &saved, &synced))
	m.username.SetValue("u@example.com")
	m.password.SetValue("pw")

	tm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = tm.(model)
	if m.phase != phaseVerifying {
		t.Fatalf("phase after submit = %v", m.phase)
	}
	if _, ok := cmd().(verifiedMsg); !ok {
		t.Fatal("submit should produce a verifiedMsg")
	}

	tm, cmd = m.Update(verifiedMsg{})
	m = tm.(model)
	if !saved || m.phase != phaseSyncing {
		t.Fatalf("after verify: saved=%v phase=%v", saved, m.phase)
	}
	sm, ok := cmd().(syncedMsg)
	if !ok || sm.err != nil || !synced {
		t.Fatalf("sync step: msg=%#v synced=%v", sm, synced)
	}

	_, cmd = m.Update(syncedMsg{})
	if cmd == nil {
		t.Fatal("successful sync should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", cmd())
	}
}

// TestOnboardingInvalidCredentials keeps the user on the form without saving.
func TestOnboardingInvalidCredentials(t *testing.T) {
	t.Parallel()
	var saved bool
	m := newModel(testDeps(ErrInvalidCredentials, nil, nil, &saved, nil))
	m.phase = phaseVerifying

	tm, _ := m.Update(verifiedMsg{err: ErrInvalidCredentials})
	m = tm.(model)
	if m.phase != phaseForm {
		t.Fatalf("phase = %v, want form", m.phase)
	}
	if !strings.Contains(m.errMsg, "Incorrect") {
		t.Fatalf("errMsg = %q", m.errMsg)
	}
	if saved {
		t.Fatal("must not save on invalid credentials")
	}
}

// TestOnboardingEmptyFieldsBlocked verifies submit requires both fields.
func TestOnboardingEmptyFieldsBlocked(t *testing.T) {
	t.Parallel()
	m := newModel(testDeps(nil, nil, nil, nil, nil))
	tm, cmd := m.submit()
	m = tm.(model)
	if m.phase != phaseForm || m.errMsg == "" || cmd != nil {
		t.Fatalf("empty submit: phase=%v err=%q hasCmd=%v", m.phase, m.errMsg, cmd != nil)
	}
}

// TestOnboardingSyncFailureReturnsToForm verifies a failed sync is recoverable.
func TestOnboardingSyncFailureReturnsToForm(t *testing.T) {
	t.Parallel()
	m := newModel(testDeps(nil, nil, errors.New("network down"), nil, nil))
	m.phase = phaseSyncing

	tm, _ := m.Update(syncedMsg{err: errors.New("network down")})
	m = tm.(model)
	if m.phase != phaseForm || !strings.Contains(m.errMsg, "sync failed") {
		t.Fatalf("sync failure: phase=%v err=%q", m.phase, m.errMsg)
	}
}

// TestOnboardingTabCyclesFocus verifies focus moves between (and wraps around)
// the two fields.
func TestOnboardingTabCyclesFocus(t *testing.T) {
	t.Parallel()
	m := newModel(testDeps(nil, nil, nil, nil, nil))
	if m.focus != focusUsername {
		t.Fatalf("initial focus = %d", m.focus)
	}
	tm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = tm.(model)
	if m.focus != focusPassword {
		t.Fatalf("focus after tab = %d", m.focus)
	}
	tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = tm.(model)
	if m.focus != focusUsername {
		t.Fatalf("focus after two tabs = %d, want wrap to username", m.focus)
	}
}

// TestOnboardingViewRenders verifies the modal composites over the background
// without panicking and shows the key elements.
func TestOnboardingViewRenders(t *testing.T) {
	t.Parallel()
	m := newModel(testDeps(nil, nil, nil, nil, nil))
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = tm.(model)

	content := m.View().Content
	for _, want := range []string{"Set up thinx", "New here?", "Things Cloud", "Cultured Code", "enter: submit", "ctrl+c: quit"} {
		if !strings.Contains(content, want) {
			t.Fatalf("view missing %q", want)
		}
	}

	// While syncing, the status line replaces the button.
	m.phase = phaseSyncing
	if !strings.Contains(m.View().Content, "Initial sync") {
		t.Fatal("syncing view should show the initial-sync status")
	}
}

// TestOnboardingEscQuits verifies the quit key works from the form.
func TestOnboardingEscQuits(t *testing.T) {
	t.Parallel()
	m := newModel(testDeps(nil, nil, nil, nil, nil))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("esc should quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", cmd())
	}
}
