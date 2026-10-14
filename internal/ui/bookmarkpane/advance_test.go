package bookmarkpane

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

func newAdvancePane(t *testing.T, runner *test.CommandRunner) *Model {
	t.Helper()
	m := New(test.NewTestContext(runner))
	m.currentCommitID = "abc123"
	return m
}

func TestAdvanceClosestToSelectedRevision(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkAdvance("abc123"))
	runner.Expect(jj.BookmarkListAll()) // the pane reloads after the refresh
	defer runner.Verify()
	m := newAdvancePane(t, runner)

	cmd := m.Advance(false)
	require.NotNil(t, cmd)
	test.SimulateModel(m, cmd)
}

func TestAdvanceSelectedLocalBookmarksByName(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeat;.;true;false;false;false;def456\n"))
	runner.Expect(jj.BookmarkAdvance("abc123", "feat"))
	defer runner.Verify()
	m := newAdvancePane(t, runner)
	test.SimulateModel(m, m.loadRows)
	m.selected["feat"] = true

	test.SimulateModel(m, m.Advance(false))
}

func TestAdvanceWithoutRevisionContextDoesNothing(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	require.Nil(t, m.Advance(false))
}

func TestAdvanceToWorkingCopyAsksBeforePushing(t *testing.T) {
	m := newAdvancePane(t, test.NewTestCommandRunner(t))
	m.Advance(true)
	require.NotNil(t, m.confirmation, "advance + push waits for a yes")
}
