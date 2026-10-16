package bookmarkpane

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

const journalRows = "feat;.;true;false;false;false;f1\n" +
	"feat;origin;true;true;false;false;f0\n" +
	"main;.;true;false;false;false;a2\n" +
	"main;origin;true;true;false;false;a1\n" +
	"spike;.;true;false;false;false;s1\n"

func newJournalPane(t *testing.T) *Model {
	t.Helper()
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkListAll()).SetOutput([]byte(journalRows))
	m := New(test.NewTestContext(runner))
	m.EnableJournalView([]string{"main"})
	test.SimulateModel(m, m.loadRows)
	runner.Verify()
	return m
}

func TestJournalViewPinsTrunkFirst(t *testing.T) {
	m := newJournalPane(t)
	require.Equal(t, "main", m.tree.Items[0].Bookmark.Name)
}

func TestJournalViewRowsShowSyncState(t *testing.T) {
	m := newJournalPane(t)
	m.SetSync(map[string]Sync{
		SyncKey("main", "origin"): {Ahead: 2},
		SyncKey("feat", "origin"): {Ahead: 1, Behind: 3},
	})
	out := ansi.Strip(test.RenderImmediate(m, 50, 8))
	require.Contains(t, out, "trunk main  origin ↑2")
	require.Contains(t, out, "feat  origin ⇅1/3")
	require.Contains(t, out, "spike  local only")
	require.NotContains(t, out, "Bookmarks", "no heading")
	require.NotContains(t, out, "Remotes:", "remote filter hidden while unfocused")
}

func TestJournalViewRowsClipToWidth(t *testing.T) {
	m := newJournalPane(t)
	m.SetSync(map[string]Sync{SyncKey("main", "origin"): {Ahead: 12}})
	m.toggleExpandSelected()
	out := ansi.Strip(test.RenderImmediate(m, 24, 6))
	require.Contains(t, out, "…")
}

func TestSyncWords(t *testing.T) {
	require.Equal(t, "in sync", syncWords(Sync{}))
	require.Equal(t, "local 2 ahead, push to sync", syncWords(Sync{Ahead: 2}))
	require.Equal(t, "local 3 behind, remote is newer", syncWords(Sync{Behind: 3}))
	require.Equal(t, "diverged: 1 local, 3 remote", syncWords(Sync{Ahead: 1, Behind: 3}))
}
