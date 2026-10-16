package statusinfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

func TestParsers(t *testing.T) {
	require.Equal(t, "xmqwnktn", FirstLine([]byte("xmqwnktn\n")))
	require.Equal(t, "", FirstLine(nil))
	require.Equal(t, "main", FirstBookmark([]byte("main,release\n")))
	require.Equal(t, "", FirstBookmark([]byte("\n")))
	require.Equal(t, 3, Count([]byte("xxx")))
	require.Equal(t, 0, Count(nil))
}

func TestFormatAge(t *testing.T) {
	now := time.Date(2026, 9, 4, 17, 0, 0, 0, time.UTC)
	require.Equal(t, "never", FormatAge(now, time.Time{}))
	require.Equal(t, "just now", FormatAge(now, now.Add(-20*time.Second)))
	require.Equal(t, "5m ago", FormatAge(now, now.Add(-5*time.Minute)))
	require.Equal(t, "3h ago", FormatAge(now, now.Add(-3*time.Hour)))
	require.Equal(t, "2d ago", FormatAge(now, now.Add(-49*time.Hour)))
}

func TestLines(t *testing.T) {
	now := time.Date(2026, 9, 4, 17, 0, 0, 0, time.UTC)
	info := Info{Repo: "jjjournal", ChangeID: "rkvnppyv", Bookmark: "main", Remote: "origin",
		Ahead: 2, Behind: 0, HasRemote: true, FetchedAt: now.Add(-3 * time.Hour), Revset: "all()", Mode: "journal"}
	lines := info.Lines(now)
	require.Equal(t, "jjjournal  @ rkvnppyv", lines[0])
	require.Equal(t, "main ↑2 ↓0 origin · fetched 3h ago", lines[1])
	require.Equal(t, "journal · all()", lines[2])

	info.HasRemote = false
	require.Equal(t, "main · not on origin · fetched 3h ago", info.Lines(now)[1])
	info.Bookmark = ""
	require.Equal(t, "(no bookmark) · fetched 3h ago", info.Lines(now)[1])
}

func TestFetchTime(t *testing.T) {
	root := t.TempDir()
	require.True(t, FetchTime(root).IsZero())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "FETCH_HEAD"), []byte("x"), 0o644))
	require.False(t, FetchTime(root).IsZero())
}

// cannedRunner answers by the revset it is asked about.
type cannedRunner struct {
	appContext.CommandRunner
	byRevset map[string]string
}

func (c cannedRunner) RunCommandImmediate(args []string) ([]byte, error) {
	for i, a := range args {
		if a == "-r" {
			if out, ok := c.byRevset[args[i+1]]; ok {
				return []byte(out), nil
			}
			return nil, os.ErrNotExist
		}
	}
	return nil, os.ErrNotExist
}

func TestGather(t *testing.T) {
	runner := cannedRunner{byRevset: map[string]string{
		"@":                        "rkvnppyv",
		"heads(::@ & bookmarks())": "main",
		"main@origin..main":        "xx",
		"main..main@origin":        "",
	}}
	root := t.TempDir()
	info := Gather(runner, root, "origin", "all()", "journal", time.Now())
	require.Equal(t, "rkvnppyv", info.ChangeID)
	require.Equal(t, "main", info.Bookmark)
	require.True(t, info.HasRemote)
	require.Equal(t, 2, info.Ahead)
	require.Equal(t, 0, info.Behind)
	require.Equal(t, "journal", info.Mode)
	require.Equal(t, "all()", info.Revset)
	require.Equal(t, filepath.Base(root), info.Repo, "repo is the directory basename")

	noRemote := Gather(cannedRunner{byRevset: map[string]string{"@": "abc", "heads(::@ & bookmarks())": "feature"}}, t.TempDir(), "origin", "all()", "graph", time.Now())
	require.Equal(t, "feature", noRemote.Bookmark)
	require.False(t, noRemote.HasRemote, "count query failed → ahead/behind unknown")
}

func TestNormalizeRemoteURL(t *testing.T) {
	for _, u := range []string{
		"git@github.com:a/b.git",
		"ssh://git@github.com/a/b",
		"https://github.com/a/b/",
		"github.com:a/b",
		"https://user@GitHub.com/a/b.git",
	} {
		require.Equal(t, "github.com/a/b", normalizeRemoteURL(u), u)
	}
}

func TestParseRemoteList(t *testing.T) {
	require.Equal(t, map[string]string{
		"origin":   "git@github.com:a/b.git",
		"upstream": "https://github.com/c/d",
	}, ParseRemoteList([]byte("origin git@github.com:a/b.git\nupstream https://github.com/c/d")))
	require.Empty(t, ParseRemoteList(nil))
}

func TestRemoteFetchTimes(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	head := filepath.Join(root, ".git", "FETCH_HEAD")
	require.NoError(t, os.WriteFile(head, []byte("7d1e6d39\tnot-for-merge\tbranch 'main' of github.com:a/b\n"), 0o644))
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	require.NoError(t, os.Chtimes(head, at, at))

	got := RemoteFetchTimes(root, map[string]string{
		"origin":   "git@github.com:a/b.git",
		"upstream": "git@github.com:c/d.git",
	})
	require.Len(t, got, 1)
	require.True(t, got["origin"].Equal(at))
}

func TestRemoteFetchTimesNoFetchHead(t *testing.T) {
	require.Empty(t, RemoteFetchTimes(t.TempDir(), map[string]string{"origin": "git@github.com:a/b.git"}))
}

func TestNormalizeRemoteURLWithPort(t *testing.T) {
	require.Equal(t, normalizeRemoteURL("ssh://host:22/x"), normalizeRemoteURL("ssh://git@host:22/x.git"))
}
