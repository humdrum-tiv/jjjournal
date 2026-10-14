package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

func newTestShell(hidden ...string) *Shell {
	s := New(&appContext.MainContext{}, &jjj.Config{
		Shell: jjj.ShellConfig{Enabled: true, LeftWidth: 0.34, NarrowColumns: 120, NotesRows: 8, Hidden: hidden},
	})
	s.normalMode = func() bool { return true }
	return s
}

func names(scopes []common.Scope) []keybindings.ScopeName {
	out := make([]keybindings.ScopeName, len(scopes))
	for i, sc := range scopes {
		out[i] = sc.Name
	}
	return out
}

func TestFocusIntents(t *testing.T) {
	s := newTestShell()

	_, handled := s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelDetail)})
	require.True(t, handled)
	require.Equal(t, PanelDetail, s.focus)

	s.handleIntent(intents.ShellCyclePanel{Delta: 1})
	require.Equal(t, Cycle(s.present(), PanelDetail, 1), s.focus)

	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelJournal)})
	s.handleIntent(intents.ShellZoom{Delta: 1})
	require.Equal(t, ZoomColumn, s.zoom)
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelDetail)})
	require.Equal(t, ZoomNone, s.zoom, "moving focus resets zoom")
}

func TestHiddenPanelIsNotFocusable(t *testing.T) {
	s := newTestShell("status")
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelStatus)})
	require.Equal(t, PanelJournal, s.focus, "hidden panel ignored")
	require.False(t, s.present()[PanelStatus])
	for i := 0; i < 10; i++ {
		s.handleIntent(intents.ShellCyclePanel{Delta: 1})
		require.NotEqual(t, PanelStatus, s.focus)
	}
}

func TestJournalAndDetailCannotBeHidden(t *testing.T) {
	s := newTestShell("journal", "detail")
	require.True(t, s.present()[PanelJournal])
	require.True(t, s.present()[PanelDetail])
}

func TestToggleJournalFlipsModeAndRefreshes(t *testing.T) {
	s := newTestShell()
	before := s.journalMode.Load()
	cmd, handled := s.handleIntent(intents.ShellToggleJournal{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	require.Equal(t, !before, s.journalMode.Load())
}

func TestJumpDayWithoutSourceIsConsumedButInert(t *testing.T) {
	s := newTestShell()
	cmd, handled := s.handleIntent(intents.ShellJumpDay{Forward: true})
	require.True(t, handled)
	require.Nil(t, cmd)

	s.focus = PanelDetail
	_, handled = s.handleIntent(intents.ShellJumpDay{Forward: true})
	require.False(t, handled, "day jumps belong to the Journal panel")
}

func TestScopesJournalFocused(t *testing.T) {
	s := newTestShell()
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjShell}, names(s.Scopes()))
	sc := s.Scopes()[0]
	require.Equal(t, common.LeakAll, sc.Leak, "revisions stay reachable")
	require.True(t, sc.Global, "visible behind a LeakGlobal panel scope")
}

func TestScopesDetailFocusedBlocksRevisions(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelDetail)
	scopes := s.Scopes()
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjDetail, actions.ScopeJjjShell}, names(scopes))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak)
}

func TestScopesEmptyOutsideNormalMode(t *testing.T) {
	s := newTestShell()
	s.normalMode = func() bool { return false }
	require.Empty(t, s.Scopes())
}

func TestScopesEmptyWhenShellDisabled(t *testing.T) {
	s := newTestShell()
	s.cfg.Shell.Enabled = false
	require.Empty(t, s.Scopes())
}

func TestDetailScrollIntentIsHandled(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelDetail)
	_, handled := s.handleIntent(intents.PanelScroll{Delta: 1})
	require.True(t, handled)
	_, handled = s.handleIntent(intents.PanelScroll{Delta: 1, Half: true})
	require.True(t, handled)
}

func TestSelectionEmptyUnlessAPanelProvidesOne(t *testing.T) {
	s := newTestShell()
	require.Nil(t, s.Selection().Highlighted)
}

func TestUpdateDoesNotForwardIntents(t *testing.T) {
	s := newTestShell()
	rec := &recordingPanel{}
	s.register(PanelNotes, rec)

	_, handled := s.Update(intents.Cancel{})
	require.False(t, handled)
	require.Empty(t, rec.msgs, "intents must never reach hosted models")

	_, _ = s.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Empty(t, rec.msgs, "keys must never reach hosted models")

	_, _ = s.Update(common.RefreshMsg{})
	require.Len(t, rec.msgs, 1)
}

func TestCloseViewReturnsFocusToJournalFromHostedPanels(t *testing.T) {
	s := newTestShell()
	rec := &recordingPanel{}
	s.register(PanelFiles, rec)
	s.setFocus(PanelFiles)

	_, handled := s.Update(common.CloseViewMsg{})
	require.False(t, handled, "upstream still sees the close")
	require.Equal(t, PanelJournal, s.focus)
}

// recordingPanel is a minimal panel that records forwarded messages.
type recordingPanel struct{ msgs []tea.Msg }

func (r *recordingPanel) title() string                           { return "Rec" }
func (r *recordingPanel) init() tea.Cmd                           { return nil }
func (r *recordingPanel) update(msg tea.Msg) tea.Cmd              { r.msgs = append(r.msgs, msg); return nil }
func (r *recordingPanel) draw(*render.DisplayContext, layout.Box) {}
func (r *recordingPanel) scopes() []common.Scope                  { return nil }
func (r *recordingPanel) setFocused(bool)                         {}
func (r *recordingPanel) editing() bool                           { return false }

func TestNotesRegisteredAndScoped(t *testing.T) {
	s := newTestShell()
	require.True(t, s.present()[PanelNotes])
	s.setFocus(PanelNotes)
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjNotes, actions.ScopeJjjShell}, names(s.Scopes()))
	_, handled := s.handleIntent(intents.PanelScroll{Delta: 1})
	require.True(t, handled)
	_, handled = s.handleIntent(intents.NotesDescribe{})
	require.True(t, handled, "consumed even without a core")
}

func TestScopesDropShellWhileEditing(t *testing.T) {
	s := newTestShell()
	rec := &editingPanel{}
	s.register(PanelFiles, rec)
	s.setFocus(PanelFiles)
	rec.isEditing = true
	for _, sc := range s.Scopes() {
		require.NotEqual(t, actions.ScopeJjjShell, string(sc.Name), "shell keys must not steal input")
	}
}

func TestFilesFocusedBlocksRevisions(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelFiles)
	scopes := s.Scopes()
	require.Equal(t, actions.ScopeJjjFiles, string(scopes[0].Name))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak)
	require.Equal(t, actions.ScopeJjjShell, string(scopes[len(scopes)-1].Name))
}

type editingPanel struct {
	recordingPanel
	isEditing bool
}

func (e *editingPanel) editing() bool { return e.isEditing }
func (e *editingPanel) scopes() []common.Scope {
	return []common.Scope{{Name: "fake.filter", Leak: common.LeakNone, Handler: handler{}}}
}

func TestOpenSquashFromFilesFollowsToJournal(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelFiles)
	_, handled := s.Update(intents.OpenSquash{})
	require.False(t, handled)
	require.Equal(t, PanelJournal, s.focus)
}
