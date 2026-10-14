package notes

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
)

func TestLinesNoDescription(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.text, m.loaded = "  \n", true
	require.Equal(t, []string{"(no description)"}, m.Lines(40))
}

func TestLinesWrapsToWidth(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.text, m.loaded = "feat: subject\n\nthe body is long enough to wrap\n", true
	lines := m.Lines(12)
	require.Equal(t, "feat:", lines[0][:5])
	for _, l := range lines {
		require.LessOrEqual(t, len(l), 12, "line %q", l)
	}
}

func TestSelectionChangeLoadsDescription(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.GetDescription("abc")).SetOutput([]byte("subject\n"))
	defer runner.Verify()
	m := New(test.NewTestContext(runner))

	cmd := m.Update(common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: "abc", CommitId: "1"}})
	require.NotNil(t, cmd)
	m.Update(drainDebounced(t, cmd))

	require.Equal(t, "abc", m.ChangeID())
	require.Equal(t, []string{"subject"}, m.Lines(40))
}

func TestFileSelectionKeepsRevision(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.changeID = "abc"
	cmd := m.Update(common.SelectionChangedMsg{Item: common.SelectedFile{ChangeId: "abc", File: jj.NewFileName("a")}})
	require.Nil(t, cmd)
	require.Equal(t, "abc", m.ChangeID())
}

func TestStaleDescriptionIgnored(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.seq = 2
	m.Update(descMsg{changeID: "old", text: "stale", seq: 1})
	require.False(t, m.loaded)
}

func TestDescriptionErrorIsShown(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.changeID, m.seq = "abc", 1
	m.Update(descMsg{changeID: "abc", seq: 1, err: errors.New("boom")})
	lines := m.Lines(40)
	require.Len(t, lines, 1)
	require.True(t, strings.HasPrefix(lines[0], "(description unavailable:"), "got %q", lines[0])
}

func TestRevisionSwitchClearsLoaded(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.changeID, m.text, m.loaded = "old", "old text\n", true
	m.Update(common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: "new"}})
	require.False(t, m.loaded)
}

// drainDebounced runs a (debounced) command and returns its message.
func drainDebounced(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	return cmd()
}
