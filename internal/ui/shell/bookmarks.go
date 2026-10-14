package shell

import (
	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// bookmarksPanel hosts upstream's bookmark pane persistently (spec §4.3).
type bookmarksPanel struct {
	s *Shell
	m *bookmarkpane.Model
}

func (p *bookmarksPanel) title() string { return "Bookmarks" }
func (p *bookmarksPanel) init() tea.Cmd { return p.m.OnShow() }

// update keeps the pane's revision context in sync, as ui.syncBookmarkPaneContext
// does for the split pane: it sorts by proximity and advances toward it.
func (p *bookmarksPanel) update(msg tea.Msg) tea.Cmd {
	p.sync()
	return p.m.Update(msg)
}

func (p *bookmarksPanel) sync() {
	if !p.s.hasCore {
		return
	}
	current := ""
	if rev := p.s.core.Revisions.SelectedRevision(); rev != nil {
		current = rev.CommitId
	}
	p.m.SyncRevisionContext(current, p.s.core.Revisions.GetCommitIds())
}

func (p *bookmarksPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *bookmarksPanel) setFocused(focused bool)                          { p.m.SetFocused(focused) }
func (p *bookmarksPanel) editing() bool                                    { return p.m.IsEditing() }

// scopes: jjj.bookmarks is LeakAll and precedes the pane's own scopes, so an
// unresolved raw key reaches the pane (the first non-LeakAll handler), not
// the shared jjj handler, which drops raw keys. While editing only the
// pane's scopes apply.
func (p *bookmarksPanel) scopes() []common.Scope {
	if p.m.IsEditing() {
		return p.m.Scopes()
	}
	return append([]common.Scope{{Name: actions.ScopeJjjBookmarks, Leak: common.LeakAll, Handler: p.s.h}}, p.m.Scopes()...)
}

func (p *bookmarksPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if it, ok := intent.(intents.BookmarkPaneAdvance); ok {
		return p.m.Advance(it.ToWorkingCopy), true
	}
	return nil, false
}
