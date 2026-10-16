package shell

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/test"
)

func TestLoadSyncCountsOnlyDifferingTrackedPairs(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkListAll()).SetOutput([]byte(
		"main;.;true;false;false;false;a2\n" +
			"main;origin;true;true;false;false;a1\n" +
			"main;git;true;true;false;false;a2\n" +
			"v1;.;true;false;false;false;b1\n" +
			"v1;origin;true;true;false;false;b1\n" +
			"spike;.;true;false;false;false;s1\n" +
			"spike;upstream;true;false;false;false;s0\n"))
	runner.Expect(jj.BookmarkOnlyIn("main", "origin", "")).SetOutput([]byte("x\nx\nx"))
	runner.Expect(jj.BookmarkOnlyIn("main", "", "origin")).SetOutput([]byte(""))
	defer runner.Verify()

	s := newRunnerShell(t, runner)
	p := s.panels[PanelBookmarks].(*bookmarksPanel)
	msg := p.loadSync()().(syncLoadedMsg)

	require.Equal(t, map[string]bookmarkpane.Sync{
		"main@origin": {Ahead: 3},
		"v1@origin":   {},
	}, msg.sync, "git remote and untracked pairs are skipped; equal pairs need no jj call")
	require.Equal(t, p.syncSeq, msg.seq)
}
