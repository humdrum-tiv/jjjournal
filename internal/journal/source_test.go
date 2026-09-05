package journal

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/idursun/jjui/internal/jj"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

// cannedRunner answers the two queries NewSource makes: the row query (its
// template carries the jjui marker) and the timestamp query.
type cannedRunner struct {
	appContext.CommandRunner
	rows, stamps string
	err          error
	calls        [][]string
}

func (c *cannedRunner) RunCommandImmediate(args []string) ([]byte, error) {
	c.calls = append(c.calls, args)
	if c.err != nil {
		return nil, c.err
	}
	for i, a := range args {
		if a == "-T" && strings.Contains(args[i+1], jj.JJUIPrefix) {
			return []byte(strings.TrimRight(c.rows, "\n")), nil // RunCommandImmediate trims newlines
		}
	}
	return []byte(c.stamps), nil
}

// marker mimics one `jj log --color always` row start: the id marker, then
// the styled short change id. The SGR sequence matters: the parser takes the
// commit id from the text up to the next style boundary (row_line.go:44-47),
// exactly as real output places a coloured change id right after the marker.
func marker(changeID, commitID string) string {
	return jj.JJUIPrefix + changeID + jj.JJUIPrefix + commitID + " \x1b[35m" + changeID + "\x1b[0m "
}

func TestNewSourceArrangesRowsAndServesOnce(t *testing.T) {
	runner := &cannedRunner{
		rows: marker("wc", "0001") + "17:00 (working copy)\n" +
			marker("aa", "0002") + "10:11 older same day\n" +
			marker("bb", "0003") + "16:30 newest\n",
		stamps: "0001\t2026-09-04T17:00:00-0700\t@\n" +
			"0002\t2026-09-04T10:11:00-0700\t-\n" +
			"0003\t2026-09-04T16:30:00-0700\t-\n",
	}
	src, err := NewSource(runner, "all()", RowTemplateJournal, 5000, Options{Now: time.Date(2026, 9, 4, 17, 30, 0, 0, time.FixedZone("PDT", -7*3600))})
	require.NoError(t, err)

	batch := src.RequestMore()
	require.False(t, batch.HasMore)
	require.Len(t, batch.Rows, 3)
	require.Equal(t, "0001", batch.Rows[0].Commit.CommitId, "working copy first")
	require.True(t, batch.Rows[0].Commit.IsWorkingCopy)
	require.Equal(t, "0003", batch.Rows[1].Commit.CommitId)
	require.Equal(t, "0002", batch.Rows[2].Commit.CommitId)
	require.Equal(t, []int{0}, src.HeaderIndices())
	require.Contains(t, lineText(batch.Rows[0].Lines[1]), "Today · Fri 2026-09-04 · 3 changes")

	require.Empty(t, src.RequestMore().Rows, "second request is empty")
	src.Close()

	require.Len(t, runner.calls, 2)
	require.Contains(t, runner.calls[0], "--no-graph")
	require.NotContains(t, runner.calls[0], "--ignore-working-copy", "row query must snapshot the working copy")
	require.Contains(t, runner.calls[1], "--ignore-working-copy")
}

func TestNewSourceEmptyRevsetIsEOF(t *testing.T) {
	src, err := NewSource(&cannedRunner{rows: "", stamps: ""}, "none()", RowTemplateJournal, 0, Options{})
	require.Nil(t, src)
	require.ErrorIs(t, err, io.EOF)
}

func TestNewSourcePropagatesJJErrors(t *testing.T) {
	src, err := NewSource(&cannedRunner{err: errors.New("Error: Revision `nope` doesn't exist")}, "nope", RowTemplateJournal, 0, Options{})
	require.Nil(t, src)
	require.EqualError(t, err, "Error: Revision `nope` doesn't exist")
}

func TestLogArgs(t *testing.T) {
	args := LogArgs("all()", 5000, RowTemplateJournal)
	require.Equal(t, []string{"log", "--no-graph", "--color", "always", "--quiet", "-r", "all()", "-T", jj.LogTemplate(RowTemplateJournal), "--limit", "5000"}, args)
	_ = context.Background
}
