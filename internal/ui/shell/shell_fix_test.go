package shell

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	"github.com/idursun/jjui/internal/ui/intents"
	bookmarkop "github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/test"
)

var revA = common.SelectedRevision{ChangeId: "aaa", CommitId: "1"}

// newRunnerShell is a shell whose context runs jj through runner, with a
// known 200x40 window (Files outer box: x 0-67, y 5-12; Journal below).
func newRunnerShell(t *testing.T, runner *test.CommandRunner, hidden ...string) *Shell {
	t.Helper()
	s := New(test.NewTestContext(runner), &jjj.Config{
		Shell: jjj.ShellConfig{Enabled: true, LeftWidth: 0.34, NarrowColumns: 120, NotesRows: 8, Hidden: hidden},
	})
	s.normalMode = func() bool { return true }
	s.width, s.height = 200, 40
	return s
}

// run executes cmd and every command nested in batches, returning the
// leaf messages.
func run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, run(c)...)
		}
		return out
	}
	if msg == nil {
		return nil
	}
	return []tea.Msg{msg}
}

// loadFiles loads revA (one file, a.go) into the Files panel.
func loadFiles(t *testing.T, s *Shell, runner *test.CommandRunner) *filesPanel {
	t.Helper()
	runner.Expect(jj.Snapshot())
	runner.Expect(jj.Status("aaa")).SetOutput([]byte("false $\nM a.go\n"))
	fp := s.panels[PanelFiles].(*filesPanel)
	for _, msg := range run(fp.m.Update(common.SelectionChangedMsg{Item: revA})) {
		for _, m2 := range run(fp.m.Update(msg)) {
			fp.m.Update(m2)
		}
	}
	_, ok := fp.m.Current()
	require.True(t, ok, "a.go loaded")
	return fp
}

// F1: split "Yes" closes with common.Close, not confirmation.Close.
func TestSplitYesInFilesDisarmsConfirmation(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	s := newRunnerShell(t, runner)
	fp := loadFiles(t, s, runner)
	s.setFocus(PanelFiles)

	_, handled := common.RouteIntent(s.Scopes(), intents.DetailsSplit{})
	require.True(t, handled)
	require.True(t, fp.editing(), "split confirmation open")

	runner.Expect(jj.Split("aaa", []jj.FileName{jj.NewFileName("a.go")}, false, false))
	yes, handled := common.RouteIntent(s.Scopes(), intents.Apply{})
	require.True(t, handled)
	for _, msg := range run(yes) {
		if _, ok := msg.(common.CloseViewMsg); ok {
			s.Update(msg)
		}
	}
	runner.Verify()
	require.False(t, fp.editing(), "the close disarms the hosted confirmation")
	require.Equal(t, PanelJournal, s.focus)

	require.True(t, s.setFocus(PanelFiles))
	again, _ := common.RouteIntent(s.Scopes(), intents.Apply{})
	require.Nil(t, again, "a second y runs nothing")
}

// F2: clicks in upstream's inline details or split bookmark pane carry
// indices into upstream's lists and must stay upstream's.
func TestFileClickOutsideFilesPanelStaysUpstream(t *testing.T) {
	s := newRunnerShell(t, test.NewTestCommandRunner(t))
	rec := &recordingPanel{}
	s.register(PanelFiles, rec)

	s.Update(tea.MouseClickMsg{X: 10, Y: 30}) // inside Journal
	_, handled := s.Update(details.FileClickedMsg{Index: 3})
	require.False(t, handled)
	require.Equal(t, PanelJournal, s.focus)
	require.Empty(t, rec.msgs, "not forwarded to Files either")

	s.Update(tea.MouseClickMsg{X: 10, Y: 7}) // inside Files
	_, handled = s.Update(details.FileClickedMsg{Index: 0})
	require.True(t, handled)
	require.Equal(t, PanelFiles, s.focus)
	require.Len(t, rec.msgs, 1)
}

func TestClicksIgnoredOutsideNormalMode(t *testing.T) {
	s := newRunnerShell(t, test.NewTestCommandRunner(t))
	rec := &recordingPanel{}
	s.register(PanelBookmarks, rec)
	s.normalMode = func() bool { return false }

	s.Update(tea.MouseClickMsg{X: 10, Y: 15}) // inside Bookmarks
	_, handled := s.Update(bookmarkpane.ItemClickedMsg{Index: 1})
	require.False(t, handled)
	require.Equal(t, PanelJournal, s.focus)
	require.Empty(t, rec.msgs)
}

// F3: bookmark_pane binds tab to ui.focus_next_pane.
func TestTabInBookmarksMovesFocus(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	scopes := s.Scopes()
	start := slices.IndexFunc(scopes, func(sc common.Scope) bool { return sc.Name == actions.ScopeBookmarkPane })
	require.GreaterOrEqual(t, start, 0)

	_, handled := common.RouteIntent(scopes[start:], intents.FocusNextPane{})
	require.True(t, handled)
	require.Equal(t, Cycle(s.focusable(), PanelBookmarks, 1), s.focus)
	require.NotEqual(t, PanelBookmarks, s.focus)
}

// F4a: focus cannot leave a panel that is editing.
func TestFocusRefusedWhileEditing(t *testing.T) {
	s := newTestShell()
	ed := &editingPanel{}
	s.register(PanelFiles, ed)
	s.setFocus(PanelFiles)
	ed.isEditing = true

	s.Update(focusPanelMsg{id: PanelBookmarks})
	require.Equal(t, PanelFiles, s.focus)

	_, handled := s.Update(revisions.ItemClickedMsg{Index: 2})
	require.True(t, handled, "swallowed: the cursor must not move under an open confirmation")
	require.Equal(t, PanelFiles, s.focus)

	ed.isEditing = false
	s.Update(focusPanelMsg{id: PanelBookmarks})
	require.Equal(t, PanelBookmarks, s.focus)
}

// F4b: confirmation messages reach only the focused panel.
func TestConfirmationMessagesOnlyReachFocusedPanel(t *testing.T) {
	s := newTestShell()
	files, marks := &recordingPanel{}, &recordingPanel{}
	s.register(PanelFiles, files)
	s.register(PanelBookmarks, marks)
	s.setFocus(PanelFiles)

	s.Update(confirmation.SelectOptionMsg{Index: 0})
	s.Update(confirmation.CloseMsg{})
	require.Len(t, files.msgs, 2)
	require.Empty(t, marks.msgs, "a leftover bookmark confirmation must not fire")
}

// F5: hidden panels receive nothing.
func TestHiddenPanelReceivesNoMessages(t *testing.T) {
	s := newTestShell("files")
	rec := &recordingPanel{}
	s.register(PanelFiles, rec)
	s.Update(common.RefreshMsg{})
	s.Update(common.SelectionChangedMsg{Item: revA})
	require.Empty(t, rec.msgs)
}

// F5: Files not drawn (narrow mode, Journal focused) defers its load and
// loads once when it is drawn.
func TestFilesLoadDeferredUntilDrawn(t *testing.T) {
	runner := test.NewTestCommandRunner(t) // no expectations: any jj call fails
	s := newRunnerShell(t, runner, "bookmarks", "status", "notes")
	s.width = 100 // narrow: Journal focused, Files not drawn
	s.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	fp := s.panels[PanelFiles].(*filesPanel)

	for _, msg := range run(fp.m.Update(common.SelectionChangedMsg{Item: revA})) {
		require.Empty(t, run(firstCmd(s.Update(msg))), "no load while not drawn")
	}
	s.Update(common.RefreshMsg{})

	runner.Expect(jj.Snapshot())
	runner.Expect(jj.Status("aaa")).SetOutput([]byte("false $\nM a.go\n"))
	cmd, handled := s.h.HandleIntent(intents.ShellFocusPanel{Panel: int(PanelFiles)})
	require.True(t, handled)
	for _, msg := range run(cmd) {
		for _, m2 := range run(firstCmd(s.Update(msg))) {
			s.Update(m2)
		}
	}
	runner.Verify()
	_, ok := fp.m.Current()
	require.True(t, ok, "loaded on show")

	cmd, _ = s.h.HandleIntent(intents.ShellFocusPanel{Panel: int(PanelFiles)})
	require.Nil(t, cmd, "loads once")
}

func firstCmd(cmd tea.Cmd, _ bool) tea.Cmd { return cmd }

// F6: a panel that gets no drawable box cannot take focus.
func TestUndrawnPanelNotFocusable(t *testing.T) {
	s := newRunnerShell(t, test.NewTestCommandRunner(t))
	s.cfg.Shell.NotesRows = 0
	require.False(t, s.setFocus(PanelNotes))
	for range 10 {
		s.handleIntent(intents.ShellCyclePanel{Delta: 1})
		require.NotEqual(t, PanelNotes, s.focus)
	}
}

func TestFocusFallsBackToJournalWhenPanelLosesItsBox(t *testing.T) {
	s := newRunnerShell(t, test.NewTestCommandRunner(t))
	require.True(t, s.setFocus(PanelNotes))
	s.Update(tea.WindowSizeMsg{Width: 200, Height: 12}) // right column too short for Notes
	require.Equal(t, PanelJournal, s.focus)
}

// F7: cancelling a move begun in Bookmarks returns there.
func TestCancelledMoveReturnsToBookmarks(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	s.Update(bookmarkpane.BeginMoveBookmarkMsg{Name: "main"})
	require.Equal(t, PanelJournal, s.focus)
	s.Update(common.CloseViewMsg{})
	s.Update(bookmarkop.MoveBookmarkCancelledMsg{Name: "main"})
	require.Equal(t, PanelBookmarks, s.focus)

	// a move begun from Journal (upstream's split pane) stays on Journal
	s.setFocus(PanelJournal)
	s.Update(bookmarkpane.BeginMoveBookmarkMsg{Name: "main"})
	s.Update(bookmarkop.MoveBookmarkCancelledMsg{Name: "main"})
	require.Equal(t, PanelJournal, s.focus)
}
