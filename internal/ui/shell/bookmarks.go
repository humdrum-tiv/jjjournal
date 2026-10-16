package shell

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/statusinfo"

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
	// syncSeq tags sync loads; only the latest result is applied.
	syncSeq int
}

// syncLoadedMsg carries the sync states of one load.
type syncLoadedMsg struct {
	seq     int
	sync    map[string]bookmarkpane.Sync
	fetched map[string]time.Time
}

func (p *bookmarksPanel) title() string { return "Remotes" }
func (p *bookmarksPanel) init() tea.Cmd { return tea.Batch(p.m.OnShow(), p.loadSync()) }

// loadSync compares every local bookmark with each tracked remote copy.
// Pairs on the same commit need no jj call; others cost two `jj log`s.
func (p *bookmarksPanel) loadSync() tea.Cmd {
	p.syncSeq++
	seq, ctx := p.syncSeq, p.s.ctx
	return func() tea.Msg {
		out, err := ctx.RunCommandImmediate(jj.BookmarkListAll())
		if err != nil {
			return syncLoadedMsg{seq: seq}
		}
		var fetched map[string]time.Time
		if remotes, err := ctx.RunCommandImmediate(jj.GitRemoteListReadOnly()); err == nil {
			fetched = statusinfo.RemoteFetchTimes(ctx.Location, statusinfo.ParseRemoteList(remotes))
		}
		count := func(args jj.CommandArgs) (int, bool) {
			o, err := ctx.RunCommandImmediate(args)
			if err != nil {
				return 0, false
			}
			return strings.Count(string(o), "x"), true
		}
		sync := map[string]bookmarkpane.Sync{}
		for _, b := range jj.ParseBookmarkListOutput(string(out)) {
			if b.Local == nil || !b.Local.Present {
				continue
			}
			for _, r := range b.Remotes {
				if !r.Tracked || !r.Present || r.Remote == "git" {
					continue
				}
				key := bookmarkpane.SyncKey(b.Name, r.Remote)
				if r.CommitId == b.Local.CommitId {
					sync[key] = bookmarkpane.Sync{}
					continue
				}
				ahead, ok1 := count(jj.BookmarkOnlyIn(b.Name, r.Remote, ""))
				behind, ok2 := count(jj.BookmarkOnlyIn(b.Name, "", r.Remote))
				if ok1 && ok2 {
					sync[key] = bookmarkpane.Sync{Ahead: ahead, Behind: behind}
				}
			}
		}
		return syncLoadedMsg{seq: seq, sync: sync, fetched: fetched}
	}
}

// update keeps the pane's revision context in sync, as ui.syncBookmarkPaneContext
// does for the split pane: it sorts by proximity and advances toward it.
func (p *bookmarksPanel) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg, common.UpdateRevisionsSuccessMsg:
		p.sync()
	case common.RefreshMsg:
		return tea.Batch(p.loadSync(), p.m.Update(msg))
	case syncLoadedMsg:
		if msg.seq == p.syncSeq && msg.sync != nil {
			p.m.SetSync(msg.sync)
			p.m.SetFetchTimes(msg.fetched, time.Now())
		}
		return nil
	}
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
