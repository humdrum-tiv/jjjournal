package journal

import (
	"fmt"
	"sort"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
)

// Options controls Arrange. Now anchors "Today"/"Yesterday" and the local
// zone; the styles are applied to header lines (zero values in tests).
type Options struct {
	Now       time.Time
	DayStyle  lipgloss.Style
	WeekStyle lipgloss.Style
}

// Arranged is Arrange's result: rows in display order plus the indices of the
// rows that carry a day header (the jump targets for `{` / `}`).
type Arranged struct {
	Rows          []parser.Row
	HeaderIndices []int
}

const dayKey = "2006-01-02"

// Arrange orders rows newest-first by committer time, working copy first,
// and prepends header lines: a day header on the first row of each calendar
// day and, above it, a week header on the first row of each ISO week. Rows
// with no stamp keep their relative order after the stamped rows and get no
// header. Header lines have no flags, so they are never highlighted or
// treated as the revision line.
func Arrange(rows []parser.Row, stamps map[string]Stamp, opts Options) Arranged {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	loc := opts.Now.Location()

	sorted := make([]parser.Row, len(rows))
	copy(sorted, rows)
	for i := range sorted {
		sorted[i].Previous = nil
		if sorted[i].Commit == nil {
			continue // no CommitId to key stamps by; leave it unstamped
		}
		if s, ok := stamps[sorted[i].Commit.CommitId]; ok && s.WorkingCopy {
			sorted[i].Commit.IsWorkingCopy = true
		}
	}
	// stampOf and isWorkingCopy guard against a nil Commit (jjui's parser can
	// hand back a header/continuation row with no commit) so Arrange never
	// panics; such a row just sorts as unstamped and gets no day header.
	stampOf := func(r parser.Row) (time.Time, bool) {
		if r.Commit == nil {
			return time.Time{}, false
		}
		s, ok := stamps[r.Commit.CommitId]
		return s.Time.In(loc), ok
	}
	isWorkingCopy := func(r parser.Row) bool {
		return r.Commit != nil && r.Commit.IsWorkingCopy
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if isWorkingCopy(a) != isWorkingCopy(b) {
			return isWorkingCopy(a)
		}
		at, aok := stampOf(a)
		bt, bok := stampOf(b)
		if aok != bok {
			return aok
		}
		if !aok {
			return false
		}
		return at.After(bt)
	})

	counts := map[string]int{}
	for _, r := range sorted {
		if t, ok := stampOf(r); ok {
			counts[t.Format(dayKey)]++
		}
	}

	out := Arranged{Rows: sorted}
	prevDay := ""
	prevYear, prevWeek := 0, 0
	for i := range sorted {
		t, ok := stampOf(sorted[i])
		if !ok {
			continue
		}
		day := t.Format(dayKey)
		if day == prevDay {
			continue
		}
		prependLine(&sorted[i], dayHeader(t, opts.Now, counts[day]), opts.DayStyle)
		year, week := t.ISOWeek()
		if year != prevYear || week != prevWeek {
			prependLine(&sorted[i], weekHeader(t, week), opts.WeekStyle)
		}
		out.HeaderIndices = append(out.HeaderIndices, i)
		prevDay, prevYear, prevWeek = day, year, week
	}
	return out
}

func dayHeader(t, now time.Time, count int) string {
	label := t.Format("Mon 2006-01-02")
	switch t.Format(dayKey) {
	case now.Format(dayKey):
		label = "Today · " + label
	case now.AddDate(0, 0, -1).Format(dayKey):
		label = "Yesterday · " + label
	}
	noun := "changes"
	if count == 1 {
		noun = "change"
	}
	return fmt.Sprintf("── %s · %d %s", label, count, noun)
}

func weekHeader(t time.Time, week int) string {
	monday := t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
	sunday := monday.AddDate(0, 0, 6)
	return fmt.Sprintf("══ Week %d · %s – %s", week, monday.Format("Jan 2"), sunday.Format("Jan 2"))
}

func prependLine(row *parser.Row, text string, style lipgloss.Style) {
	line := parser.NewGraphRowLine([]*screen.Segment{{Text: text, Style: style}})
	line.Flags = 0
	row.Lines = append([]*parser.GraphRowLine{&line}, row.Lines...)
}

// JumpTarget returns the header index after (forward) or before cursor, or -1.
func JumpTarget(headers []int, cursor int, forward bool) int {
	if forward {
		for _, h := range headers {
			if h > cursor {
				return h
			}
		}
		return -1
	}
	for i := len(headers) - 1; i >= 0; i-- {
		if headers[i] < cursor {
			return headers[i]
		}
	}
	return -1
}
