// Package tags is jjj's Tags tab (panel 3, key 8): your milestone tags,
// newest first, with date and subject; enter reveals a tag in Journal.
package tags

import (
	"slices"
	"strings"
	"time"

	"github.com/idursun/jjui/internal/jj"
)

// Tag is one milestone. A conflicted tag has no single target: no CommitID
// and no When.
type Tag struct {
	Name, CommitID, Subject string
	When                    time.Time
	Conflict                bool
}

// ForeignRemotes are the remotes (other than git) that appear in
// jj.BookmarkListAll output without a single tracked bookmark: projects
// you fetch from but do not work on, such as a fork's upstream.
func ForeignRemotes(bookmarkListOutput string) []string {
	tracked := map[string]bool{}
	for _, b := range jj.ParseBookmarkListOutput(bookmarkListOutput) {
		for _, r := range b.Remotes {
			if r.Remote != "git" {
				tracked[r.Remote] = tracked[r.Remote] || r.Tracked
			}
		}
	}
	var out []string
	for remote, t := range tracked {
		if !t {
			out = append(out, remote)
		}
	}
	slices.Sort(out)
	return out
}

// ParseTags reads jj.TagListJournal output. Foreign and deleted tags are
// dropped; the rest sort newest first, then by name, conflicted ones last.
func ParseTags(output string) []Tag {
	var tags []Tag
	for _, line := range strings.Split(output, "\n") {
		f := strings.SplitN(line, ";", 6)
		if len(f) < 6 || f[1] != "true" || f[4] == "true" {
			continue
		}
		t := Tag{Name: f[0], CommitID: f[2], Subject: f[5]}
		if t.CommitID == "" {
			t = Tag{Name: f[0], Conflict: true}
		} else if when, err := time.Parse("2006-01-02T15:04:05-0700", f[3]); err == nil {
			t.When = when
		}
		tags = append(tags, t)
	}
	slices.SortStableFunc(tags, func(a, b Tag) int {
		switch {
		case a.Conflict != b.Conflict:
			if a.Conflict {
				return 1
			}
			return -1
		case !a.When.Equal(b.When):
			return b.When.Compare(a.When)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return tags
}
