package files

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
)

type nopHandler struct{}

func (nopHandler) HandleIntent(intents.Intent) (tea.Cmd, bool) { return nil, false }
func (nopHandler) Update(tea.Msg) tea.Cmd                      { return nil }

var revA = common.SelectedRevision{ChangeId: "aaa", CommitId: "1"}
var revB = common.SelectedRevision{ChangeId: "bbb", CommitId: "2"}

func TestFilesIgnoresStaleRevision(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })

	require.NotNil(t, m.Update(common.SelectionChangedMsg{Item: revA}))
	require.NotNil(t, m.Update(common.SelectionChangedMsg{Item: revB}))
	// The debounced message for A arrives after B was requested: no load
	// (the test runner would fail on an unexpected jj command).
	require.Nil(t, m.Update(revisionMsg{rev: revA}))
	require.Nil(t, m.op, "nothing loaded for a superseded revision")
}

func TestFilesSameRevisionIsNoop(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	m.Update(common.SelectionChangedMsg{Item: revA})
	require.Nil(t, m.Update(common.SelectionChangedMsg{Item: revA}))
}

func TestFilesIgnoresFileSelections(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	require.Nil(t, m.Update(common.SelectionChangedMsg{Item: common.SelectedFile{ChangeId: "aaa", File: jj.NewFileName("x")}}))
}

func TestFilesLoadsOnLatestRevision(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.Snapshot())
	runner.Expect(jj.Status("aaa")).SetOutput([]byte("false $\nM a.go\n"))
	defer runner.Verify()
	m := New(test.NewTestContext(runner), func() bool { return true })

	m.Update(common.SelectionChangedMsg{Item: revA})
	cmd := m.Update(revisionMsg{rev: revA})
	require.NotNil(t, m.op)
	if cmd != nil {
		m.Update(cmd())
	}
}

func TestScopesWithoutOp(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	scopes := m.Scopes(nopHandler{})
	require.Len(t, scopes, 1)
	require.Equal(t, actions.ScopeJjjFiles, string(scopes[0].Name))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak, "never leak to revisions, even before loading")
}

func TestFileIntentsWithoutFilesAreConsumed(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	for _, it := range []intents.Intent{intents.FilesDiffEdit{}, intents.FilesOpenEditor{}, intents.FilesAnnotate{}, intents.FilesResolve{}} {
		cmd, handled := m.HandleIntent(it)
		require.True(t, handled, "%T", it)
		require.Nil(t, cmd, "%T", it)
	}
}
