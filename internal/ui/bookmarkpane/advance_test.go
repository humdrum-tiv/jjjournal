package bookmarkpane

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
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
	runner := test.NewTestCommandRunner(t)
	defer runner.Verify()
	m := newAdvancePane(t, runner)
	m.Advance(true)
	require.NotNil(t, m.confirmation, "advance + push waits for a yes")
}

func TestAdvanceThenPushSkipsPushWhenAdvanceFails(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkAdvance("@", "feat")).SetError(errors.New("x"))
	defer runner.Verify() // no push expectation: a push would fail the runner
	ctx := test.NewTestContext(runner)

	msg := advanceThenPush(ctx, jj.BookmarkAdvance("@", "feat"), []string{"--bookmark", "feat"})()
	done, ok := msg.(common.CommandCompletedMsg)
	require.True(t, ok)
	require.Error(t, done.Err)
}

func TestAdvanceThenPushPushesAfterSuccess(t *testing.T) {
	for name, flags := range map[string][]string{"selected": {"--bookmark", "feat"}, "none": nil} {
		t.Run(name, func(t *testing.T) {
			runner := test.NewTestCommandRunner(t)
			runner.Expect(jj.BookmarkAdvance("@"))
			runner.Expect(jj.GitPush(flags...))
			runner.Expect(jj.BookmarkListAll()) // the pane reloads after the refresh
			defer runner.Verify()
			ctx := test.NewTestContext(runner)

			msg := advanceThenPush(ctx, jj.BookmarkAdvance("@"), flags)()
			batch, ok := msg.(tea.BatchMsg)
			require.True(t, ok)
			for _, c := range batch {
				test.SimulateModel(New(ctx), c)
			}
		})
	}
}
