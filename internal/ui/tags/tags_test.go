package tags

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
)

const bookmarks = "main;.;true;false;false;false;a2\n" +
	"main;origin;true;true;false;false;a2\n" +
	"main;upstream;true;false;false;false;b1\n"

const tagRows = "v0.1;true;aa;2026-09-01T10:00:00+0000;false;first milestone\n" +
	"v0.2;true;cc;2026-10-10T10:00:00+0000;false;second milestone\n" +
	"knot;true;;;false;\n"

func loaded(t *testing.T, tagOutput string) *Model {
	t.Helper()
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkListAll()).SetOutput([]byte(bookmarks))
	runner.Expect(jj.TagListJournal([]string{"upstream"})).SetOutput([]byte(tagOutput))
	m := New(test.NewTestContext(runner))
	m.SetFocused(true)
	m.Update(m.Load()())
	runner.Verify()
	return m
}

func draw(m *Model, w, h int) string { return ansi.Strip(test.RenderImmediate(m, w, h)) }

func TestTagsLoadAndOrder(t *testing.T) {
	out := draw(loaded(t, tagRows), 60, 5)
	require.Contains(t, out, "v0.2  2026-10-10  second milestone")
	require.Contains(t, out, "v0.1  2026-09-01  first milestone")
	require.Contains(t, out, "knot  conflict")
	require.Less(t, strings.Index(out, "v0.2"), strings.Index(out, "v0.1"))
}

func TestTagsRowsClipToWidth(t *testing.T) {
	require.Contains(t, draw(loaded(t, tagRows), 20, 5), "…")
}

func TestTagsMoveClamps(t *testing.T) {
	m := loaded(t, tagRows)
	m.HandleIntent(intents.TagsMove{Delta: -1})
	sel, _ := m.Selected()
	require.Equal(t, "v0.2", sel.Name)
	for range 5 {
		m.HandleIntent(intents.TagsMove{Delta: 1})
	}
	sel, _ = m.Selected()
	require.Equal(t, "knot", sel.Name)
}

func TestTagsRevealEmitsRevealMsg(t *testing.T) {
	m := loaded(t, tagRows)
	cmd, handled := m.HandleIntent(intents.TagsReveal{})
	require.True(t, handled)
	require.Equal(t, bookmarkpane.RevealRevisionMsg{CommitID: "cc"}, cmd())
}

func TestTagsRevealConflictedIsInert(t *testing.T) {
	m := loaded(t, tagRows)
	m.HandleIntent(intents.TagsMove{Delta: 2})
	cmd, handled := m.HandleIntent(intents.TagsReveal{})
	require.True(t, handled)
	require.Nil(t, cmd)
}

func TestTagsEmpty(t *testing.T) {
	require.Contains(t, draw(loaded(t, ""), 30, 3), "No tags")
}

func TestTagsStaleLoadIgnored(t *testing.T) {
	m := loaded(t, tagRows)
	stale := loadedMsg{seq: m.seq - 1, tags: nil}
	m.Update(stale)
	_, ok := m.Selected()
	require.True(t, ok, "an older load must not replace the list")
}
