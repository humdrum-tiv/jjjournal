package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	appContext "github.com/idursun/jjui/internal/ui/context"

	"github.com/idursun/jjui/internal/jjj"
)

func newTestShell() *Shell {
	return New(&appContext.MainContext{}, &jjj.Config{
		Shell: jjj.ShellConfig{Enabled: true, LeftWidth: 0.34, NarrowColumns: 120},
	})
}

// TestHandleKey covers focus-by-digit, tab cycling, and Detail-only
// scrolling, each active and (mirrored) inactive: while s.active is false,
// HandleKey must fall through unhandled and leave focus untouched, since
// Render only sets active when the panels are actually on screen.
func TestHandleKey(t *testing.T) {
	tests := []struct {
		name        string
		startFocus  PanelID
		msg         tea.KeyPressMsg
		wantHandled bool
		wantFocus   PanelID // ignored when wantHandled is false
	}{
		{"0 focuses Detail", PanelJournal, tea.KeyPressMsg{Code: '0', Text: "0"}, true, PanelDetail},
		{"4 focuses Journal", PanelDetail, tea.KeyPressMsg{Code: '4', Text: "4"}, true, PanelJournal},
		{"tab Status->Journal", PanelStatus, tea.KeyPressMsg{Code: tea.KeyTab}, true, PanelJournal},
		{"tab Journal->Detail", PanelJournal, tea.KeyPressMsg{Code: tea.KeyTab}, true, PanelDetail},
		{"tab Detail->Status", PanelDetail, tea.KeyPressMsg{Code: tea.KeyTab}, true, PanelStatus},
		{"j scrolls Detail when focused", PanelDetail, tea.KeyPressMsg{Code: 'j', Text: "j"}, true, PanelDetail},
		{"j falls through when Journal focused", PanelJournal, tea.KeyPressMsg{Code: 'j', Text: "j"}, false, PanelJournal},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/active", func(t *testing.T) {
			s := newTestShell()
			s.focus = tt.startFocus

			_, handled := s.HandleKey(tt.msg)

			require.Equal(t, tt.wantHandled, handled)
			if tt.wantHandled {
				require.Equal(t, tt.wantFocus, s.focus)
			}
		})

		t.Run(tt.name+"/inactive", func(t *testing.T) {
			s := newTestShell()
			s.focus = tt.startFocus
			s.active = false

			cmd, handled := s.HandleKey(tt.msg)

			require.Nil(t, cmd)
			require.False(t, handled, "must not be handled while inactive")
			require.Equal(t, tt.startFocus, s.focus, "focus must not change while inactive")
		})
	}
}

// TestHandleKeyJKeepsWorkingWhenInactive covers the F3 fix: J toggles the
// list's journal/graph mode, which is visible in the stock single-list
// layout the shell falls back to under narrow_columns, so it must not be
// gated behind s.active.
func TestHandleKeyJKeepsWorkingWhenInactive(t *testing.T) {
	s := newTestShell()
	s.active = false
	before := s.journalMode.Load()

	cmd, handled := s.HandleKey(tea.KeyPressMsg{Code: 'J', Text: "J"})

	require.True(t, handled)
	require.NotNil(t, cmd)
	require.Equal(t, !before, s.journalMode.Load())
}

// TestHandleKeyBraceJumpFallsThroughWithNilSource covers {/} while inactive:
// they are evaluated before the active gate, but with no journal source
// loaded yet (or in graph mode) there is nothing to jump to, so they must
// report unhandled rather than panicking or consuming the key.
func TestHandleKeyBraceJumpFallsThroughWithNilSource(t *testing.T) {
	for _, text := range []string{"{", "}"} {
		s := newTestShell()
		s.active = false
		s.focus = PanelJournal

		_, handled := s.HandleKey(tea.KeyPressMsg{Text: text})

		require.False(t, handled, "text=%q: nil source has nothing to jump to", text)
	}
}

// TestHandleKeyDigitsAndTabStayUnhandledWhileInactive guards against widening
// the F3 fix beyond J/{/}: focus-by-digit and tab must still fall through to
// the stock layout while the shell is inactive.
func TestHandleKeyDigitsAndTabStayUnhandledWhileInactive(t *testing.T) {
	s := newTestShell()
	s.active = false
	s.focus = PanelStatus

	for _, msg := range []tea.KeyPressMsg{
		{Code: '0', Text: "0"},
		{Code: '4', Text: "4"},
		{Code: tea.KeyTab},
	} {
		_, handled := s.HandleKey(msg)
		require.False(t, handled)
		require.Equal(t, PanelStatus, s.focus, "focus must not change while inactive")
	}
}
