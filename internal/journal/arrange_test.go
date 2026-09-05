package journal

import (
	"testing"
	"time"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/stretchr/testify/require"
)

// row builds a parsed row the way the parser does for `--no-graph` output:
// one Revision|Highlightable line and an empty gutter.
func row(commitID, subject string) parser.Row {
	r := parser.NewGraphRow()
	r.Commit.CommitId = commitID
	r.Commit.ChangeId = "c" + commitID
	line := parser.NewGraphRowLine([]*screen.Segment{{Text: subject}})
	r.AddLine(&line)
	return r
}

func lineText(l *parser.GraphRowLine) string {
	s := ""
	for _, seg := range l.Segments {
		s += seg.Text
	}
	return s
}

var pdt = time.FixedZone("PDT", -7*3600)

func at(y, mo, d, h, mi int) time.Time { return time.Date(y, time.Month(mo), d, h, mi, 0, 0, pdt) }

func TestArrangeSortsNewestFirstAndInsertsHeaders(t *testing.T) {
	now := at(2026, 9, 4, 17, 0) // Friday, ISO week 36
	rows := []parser.Row{
		row("a1", "older, same day"),
		row("b2", "newest"),
		row("c3", "yesterday"),
		row("d4", "last week"),
	}
	stamps := map[string]Stamp{
		"a1": {Time: at(2026, 9, 4, 10, 11)},
		"b2": {Time: at(2026, 9, 4, 16, 30)},
		"c3": {Time: at(2026, 9, 3, 9, 0)},
		"d4": {Time: at(2026, 8, 28, 12, 0)}, // Friday of ISO week 35
	}

	got := Arrange(rows, stamps, Options{Now: now})

	ids := func() []string {
		var out []string
		for _, r := range got.Rows {
			out = append(out, r.Commit.CommitId)
		}
		return out
	}()
	require.Equal(t, []string{"b2", "a1", "c3", "d4"}, ids)
	require.Equal(t, []int{0, 2, 3}, got.HeaderIndices)

	// first row: week header, day header, then the revision line
	first := got.Rows[0].Lines
	require.Len(t, first, 3)
	require.Equal(t, "══ Week 36 · Aug 31 – Sep 6", lineText(first[0]))
	require.Equal(t, "── Today · Fri 2026-09-04 · 2 changes", lineText(first[1]))
	require.Equal(t, "newest", lineText(first[2]))
	require.Equal(t, parser.RowLineFlags(0), first[0].Flags)
	require.Equal(t, parser.RowLineFlags(0), first[1].Flags)
	require.Equal(t, parser.Revision|parser.Highlightable, first[2].Flags)

	require.Len(t, got.Rows[1].Lines, 1, "same day: no header")
	require.Equal(t, "── Yesterday · Thu 2026-09-03 · 1 change", lineText(got.Rows[2].Lines[0]))

	lastWeek := got.Rows[3].Lines
	require.Len(t, lastWeek, 3)
	require.Equal(t, "══ Week 35 · Aug 24 – Aug 30", lineText(lastWeek[0]))
	require.Equal(t, "── Fri 2026-08-28 · 1 change", lineText(lastWeek[1]))

	for _, r := range got.Rows {
		require.Nil(t, r.Previous, "journal rows carry no graph continuity")
	}
}

func TestArrangePinsWorkingCopyFirstAndMarksIt(t *testing.T) {
	rows := []parser.Row{row("new1", "a fresh commit"), row("wc00", "(working copy)")}
	stamps := map[string]Stamp{
		"new1": {Time: at(2026, 9, 4, 12, 0)},
		"wc00": {Time: at(2026, 9, 1, 8, 0), WorkingCopy: true}, // older, still first
	}
	got := Arrange(rows, stamps, Options{Now: at(2026, 9, 4, 17, 0)})
	require.Equal(t, "wc00", got.Rows[0].Commit.CommitId)
	require.True(t, got.Rows[0].Commit.IsWorkingCopy)
	require.False(t, got.Rows[1].Commit.IsWorkingCopy)
}

func TestArrangeKeepsUnstampedRowsLastInOriginalOrder(t *testing.T) {
	rows := []parser.Row{row("x", "no stamp 1"), row("y", "stamped"), row("z", "no stamp 2")}
	stamps := map[string]Stamp{"y": {Time: at(2026, 9, 4, 12, 0)}}
	got := Arrange(rows, stamps, Options{Now: at(2026, 9, 4, 17, 0)})
	require.Equal(t, "y", got.Rows[0].Commit.CommitId)
	require.Equal(t, "x", got.Rows[1].Commit.CommitId)
	require.Equal(t, "z", got.Rows[2].Commit.CommitId)
	require.Len(t, got.Rows[1].Lines, 1, "unstamped rows get no header")
}

// TestArrangeNeverPanicsOnNilCommit guards the F7 fix: a row with no Commit
// (e.g. a stray continuation/header line from the parser) must be skipped by
// the stamp lookup and the header pass rather than panicking.
func TestArrangeNeverPanicsOnNilCommit(t *testing.T) {
	stamped := row("y", "stamped")
	nilCommit := parser.Row{Lines: stamped.Lines}
	rows := []parser.Row{nilCommit, stamped}
	stamps := map[string]Stamp{"y": {Time: at(2026, 9, 4, 12, 0)}}

	require.NotPanics(t, func() {
		got := Arrange(rows, stamps, Options{Now: at(2026, 9, 4, 17, 0)})
		require.Len(t, got.Rows, 2)
	})
}

func TestJumpTarget(t *testing.T) {
	headers := []int{0, 4, 9}
	require.Equal(t, 4, JumpTarget(headers, 0, true))
	require.Equal(t, 9, JumpTarget(headers, 4, true))
	require.Equal(t, -1, JumpTarget(headers, 9, true))
	require.Equal(t, 4, JumpTarget(headers, 6, false))
	require.Equal(t, 0, JumpTarget(headers, 4, false))
	require.Equal(t, -1, JumpTarget(headers, 0, false))
	require.Equal(t, -1, JumpTarget(nil, 3, true))
}
