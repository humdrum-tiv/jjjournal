package bookmarkpane

// jjj addition (new file; see UPSTREAM.md). The journal view of the pane:
// pinned bookmarks (main) first and tagged, each local bookmark's sync state
// against its tracked remotes spelled out, no heading, and the remote filter
// line only while focused. Off unless EnableJournalView is called.

import (
	"fmt"
	"slices"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/render"
)

// Sync is a local bookmark's position relative to one remote copy:
// Ahead commits only local has, Behind commits only the remote has.
type Sync struct {
	Ahead, Behind int
}

func (s Sync) InSync() bool   { return s.Ahead == 0 && s.Behind == 0 }
func (s Sync) Diverged() bool { return s.Ahead > 0 && s.Behind > 0 }

// SyncKey indexes SetSync's map: "name@remote".
func SyncKey(name, remote string) string { return name + "@" + remote }

type journalView struct {
	pinned []string
	sync   map[string]Sync
	styles journalStyles
}

type journalStyles struct {
	pinned, inSync, ahead, behind, diverged lipgloss.Style
}

// EnableJournalView switches the pane to jjj's rows. pinned bookmarks sort
// first, in the given order, and are tagged "trunk".
func (m *Model) EnableJournalView(pinned []string) {
	p := common.DefaultPalette
	m.jjj = &journalView{
		pinned: slices.Clone(pinned),
		styles: journalStyles{
			pinned:   p.Get("picker", "", "matched", false).Bold(true),
			inSync:   p.Get("", "", "success", false),
			ahead:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
			behind:   p.Get("picker", "", "matched", false),
			diverged: p.Get("", "", "error", false),
		},
	}
	m.pinTree()
	m.applyFilters(false)
}

// SetSync replaces the sync states (keyed by SyncKey). A pair with no entry
// is drawn without a marker.
func (m *Model) SetSync(sync map[string]Sync) {
	if m.jjj != nil {
		m.jjj.sync = sync
	}
}

// pinTree moves pinned bookmarks to the front, keeping the rest in order.
func (m *Model) pinTree() {
	if m.jjj == nil || len(m.jjj.pinned) == 0 {
		return
	}
	rank := func(item bookmarkTreeItem) int {
		if i := slices.Index(m.jjj.pinned, item.Bookmark.Name); i >= 0 && !item.remoteOnly() {
			return i
		}
		return len(m.jjj.pinned)
	}
	slices.SortStableFunc(m.tree.Items, func(a, b bookmarkTreeItem) int { return rank(a) - rank(b) })
}

func (m *Model) isPinned(item bookmarkTreeItem) bool {
	return m.jjj != nil && !item.remoteOnly() && slices.Contains(m.jjj.pinned, item.Bookmark.Name)
}

// journalHeader reports whether the remote filter line is drawn: only while
// focused, or when a filter other than "all" is active.
func (m *Model) journalShowsRemotes() bool {
	return m.Focused() || m.activeRemoteFilter() != allRemoteFilter
}

func (m *Model) syncFor(name, remote string) (Sync, bool) {
	if m.jjj == nil || m.jjj.sync == nil {
		return Sync{}, false
	}
	s, ok := m.jjj.sync[SyncKey(name, remote)]
	return s, ok
}

// syncMarker is the short form: "=", "↑2", "↓3", "⇅2/1".
func (m *Model) syncMarker(s Sync) (string, lipgloss.Style) {
	st := m.jjj.styles
	switch {
	case s.Diverged():
		return fmt.Sprintf("⇅%d/%d", s.Ahead, s.Behind), st.diverged
	case s.Ahead > 0:
		return fmt.Sprintf("↑%d", s.Ahead), st.ahead
	case s.Behind > 0:
		return fmt.Sprintf("↓%d", s.Behind), st.behind
	default:
		return "=", st.inSync
	}
}

// syncWords is the long form used on expanded remote rows.
func syncWords(s Sync) string {
	switch {
	case s.Diverged():
		return fmt.Sprintf("diverged: %d local, %d remote", s.Ahead, s.Behind)
	case s.Ahead > 0:
		return fmt.Sprintf("local %d ahead, push to sync", s.Ahead)
	case s.Behind > 0:
		return fmt.Sprintf("local %d behind, remote is newer", s.Behind)
	default:
		return "in sync"
	}
}

// rowWriter appends styled text to a row until width runs out; the last
// piece that does not fit is cut with an ellipsis.
type rowWriter struct {
	tb   *render.TextBuilder
	left int
}

func (w *rowWriter) put(text string, style lipgloss.Style) *rowWriter {
	if w.left <= 0 {
		return w
	}
	if render.StringWidth(text) > w.left {
		text = ansi.Truncate(text, w.left, "…")
	}
	w.left -= render.StringWidth(text)
	w.tb.Styled(text, style)
	return w
}

func (m *Model) renderJournalTopRow(w *rowWriter, row visibleRow, group bookmarkTreeItem, node bookmarkRowNode) {
	prefix := "  "
	if row.HasChildren {
		prefix = "▸ "
		if row.Expanded {
			prefix = "▾ "
		}
	}
	w.put(prefix, m.styles.childGuide)

	switch {
	case node.IsRemote():
		w.put("remote ", m.styles.remoteBookmark).
			put(node.Name, m.styles.text).
			put("@"+node.Remote, m.styles.remoteBookmarkName)
	case m.isPinned(group):
		w.put("trunk ", m.styles.dimmed).put(node.Name, m.jjj.styles.pinned)
	default:
		w.put(node.Name, m.styles.text)
	}

	if !node.IsRemote() {
		tracked := 0
		for _, r := range group.Bookmark.Remotes {
			if !r.Tracked || r.Remote == "git" {
				continue
			}
			tracked++
			w.put("  ", m.styles.text).put(r.Remote, m.styles.dimmed)
			if s, ok := m.syncFor(group.Bookmark.Name, r.Remote); ok {
				marker, style := m.syncMarker(s)
				w.put(" ", m.styles.text).put(marker, style)
			}
		}
		if tracked == 0 && !node.Deleted {
			w.put("  ", m.styles.text).put("local only", m.styles.dimmed)
		}
	}
	if node.Deleted {
		w.put("  ", m.styles.text).put("deleted", m.styles.deleted)
	}
	if node.Conflict {
		w.put("  ", m.styles.text).put("conflict", m.styles.conflict)
	}
	if node.CommitID != "" {
		w.put("  ", m.styles.text).put(node.CommitID, m.styles.dimmed)
	}
}

func (m *Model) renderJournalRemoteRow(w *rowWriter, group bookmarkTreeItem, node bookmarkRowNode) {
	w.put("   @", m.styles.dimmed).put(node.Remote, m.styles.remoteBookmarkName).put("  ", m.styles.text)
	switch {
	case node.Deleted:
		w.put("deleted on remote", m.styles.deleted)
	case !node.Tracked:
		w.put("not tracked", m.styles.dimmed)
	case group.Bookmark.Local == nil || !group.Bookmark.Local.Present:
		w.put("no local copy", m.styles.dimmed)
	default:
		if s, ok := m.syncFor(node.Name, node.Remote); ok {
			marker, style := m.syncMarker(s)
			w.put(marker, style).put(" "+syncWords(s), m.styles.text)
		}
	}
	if node.CommitID != "" {
		w.put("  ", m.styles.text).put(node.CommitID, m.styles.dimmed)
	}
}
