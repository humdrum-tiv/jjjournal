package bookmarkpane

// jjj addition (new file; upstream PR candidate — see UPSTREAM.md).

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
)

// Advance runs `jj bookmark advance` toward the revision selected in the
// revisions list, or toward @ when toWorkingCopy is set. Multi-selected local
// bookmarks are passed by exact name; with none selected jj advances the
// closest bookmarks. toWorkingCopy also pushes, after one confirmation:
// the selected bookmarks by name, or jj's default push set when none are
// selected (bookmarks between their remote and @).
func (m *Model) Advance(toWorkingCopy bool) tea.Cmd {
	to := m.currentCommitID
	if toWorkingCopy {
		to = "@"
	}
	if to == "" {
		return nil
	}
	names := m.advanceNames()
	advance := jj.BookmarkAdvance(to, names...)
	if !toWorkingCopy {
		return m.context.RunCommand(advance, common.Refresh)
	}
	var flags []string
	for _, n := range names {
		flags = append(flags, "--bookmark", n)
	}
	what := "the closest bookmarks"
	if len(names) > 0 {
		what = strings.Join(names, ", ")
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Advance %s to @ and push?", what)},
		tea.Sequence(m.context.RunCommand(advance), m.context.RunCommand(jj.GitPush(flags...), common.Refresh)),
	)
}

// advanceNames are the multi-selected local bookmark names, in tree order.
func (m *Model) advanceNames() []string {
	if len(m.selected) == 0 {
		return nil
	}
	var names []string
	seen := map[string]bool{}
	for _, s := range m.selectionsForBookmarkOperation() {
		if s.node.IsRemote() || seen[s.bookmark.Name] {
			continue
		}
		seen[s.bookmark.Name] = true
		names = append(names, s.bookmark.Name)
	}
	return names
}
