package shell

import (
	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/files"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/notes"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/statuspanel"
)

// panel is one shell panel. update receives forwarded messages that are
// neither input nor intents (Shell.Update filters both); keys reach a panel
// only as intents through the scopes it returns.
type panel interface {
	title() string
	init() tea.Cmd
	update(msg tea.Msg) tea.Cmd
	draw(dl *render.DisplayContext, inner layout.Box)
	// scopes are innermost first; nil means the panel has no keys of its own.
	scopes() []common.Scope
	setFocused(focused bool)
	// editing is true while the panel owns raw input (a filter or a
	// confirmation); the shell then drops its own scope so keys reach it.
	editing() bool
}

// intentPanel is implemented by panels with jjj intents of their own.
type intentPanel interface {
	handleIntent(intent intents.Intent) (tea.Cmd, bool)
}

// selectionPanel is implemented by panels whose cursor is a selection
// (Files: the highlighted file drives Detail and $file).
type selectionPanel interface {
	selection() common.SelectionSnapshot
}

// handler is the common.ScopeHandler behind every jjj scope.
type handler struct{ s *Shell }

func (h handler) HandleIntent(intent intents.Intent) (tea.Cmd, bool) { return h.s.handleIntent(intent) }
func (h handler) Update(tea.Msg) tea.Cmd                             { return nil }

// journalPanel draws upstream's revset line and revisions list; its keys
// are upstream's own scopes, so it contributes none.
type journalPanel struct{ s *Shell }

func (p *journalPanel) title() string          { return "Journal" }
func (p *journalPanel) init() tea.Cmd          { return nil }
func (p *journalPanel) update(tea.Msg) tea.Cmd { return nil }
func (p *journalPanel) scopes() []common.Scope { return nil }
func (p *journalPanel) editing() bool          { return false }
func (p *journalPanel) setFocused(focused bool) {
	if p.s.hasCore {
		p.s.core.Revisions.SetFocused(focused)
	}
}
func (p *journalPanel) draw(dl *render.DisplayContext, inner layout.Box) {
	if !p.s.hasCore {
		return
	}
	rows := inner.V(layout.Fixed(1), layout.Fill(1))
	if len(rows) < 2 {
		return
	}
	p.s.core.Revset.ViewRect(dl, rows[0])
	p.s.core.RenderMain(rows[1])
}

// statusPanel hosts the three-line Status model.
type statusPanel struct {
	m *statuspanel.Model
	h handler
}

func (p *statusPanel) title() string              { return "Status" }
func (p *statusPanel) init() tea.Cmd              { return p.m.Init() }
func (p *statusPanel) update(msg tea.Msg) tea.Cmd { return p.m.Update(msg) }
func (p *statusPanel) setFocused(bool)            {}
func (p *statusPanel) editing() bool              { return false }
func (p *statusPanel) draw(dl *render.DisplayContext, inner layout.Box) {
	p.m.ViewRect(dl, inner)
}
func (p *statusPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjStatus, Leak: common.LeakGlobal, Handler: p.h}}
}

// detailPanel hosts the preview model; its scope scrolls it.
type detailPanel struct {
	m *preview.Model
	h handler
}

func (p *detailPanel) title() string              { return "Detail" }
func (p *detailPanel) init() tea.Cmd              { return p.m.Init() }
func (p *detailPanel) update(msg tea.Msg) tea.Cmd { return p.m.Update(msg) }
func (p *detailPanel) setFocused(bool)            {}
func (p *detailPanel) editing() bool              { return false }
func (p *detailPanel) draw(dl *render.DisplayContext, inner layout.Box) {
	p.m.ViewRect(dl, inner)
}
func (p *detailPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjDetail, Leak: common.LeakGlobal, Handler: p.h}}
}
func (p *detailPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	sc, ok := intent.(intents.PanelScroll)
	if !ok {
		return nil, false
	}
	switch {
	case sc.Page && sc.Delta > 0:
		return p.m.PageDown(), true
	case sc.Page:
		return p.m.PageUp(), true
	case sc.Half && sc.Delta > 0:
		return p.m.HalfPageDown(), true
	case sc.Half:
		return p.m.HalfPageUp(), true
	default:
		return p.m.Scroll(sc.Delta), true
	}
}

// notesPanel hosts the Notes model; enter opens inline describe on Journal.
type notesPanel struct {
	s *Shell
	m *notes.Model
}

func (p *notesPanel) title() string                                    { return "Notes" }
func (p *notesPanel) init() tea.Cmd                                    { return nil }
func (p *notesPanel) update(msg tea.Msg) tea.Cmd                       { return p.m.Update(msg) }
func (p *notesPanel) setFocused(bool)                                  {}
func (p *notesPanel) editing() bool                                    { return false }
func (p *notesPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *notesPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjNotes, Leak: common.LeakGlobal, Handler: p.s.h}}
}
func (p *notesPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch it := intent.(type) {
	case intents.PanelScroll:
		p.m.Scroll(it.Delta)
		return nil, true
	case intents.NotesDescribe:
		if !p.s.hasCore {
			return nil, true
		}
		rev := p.s.core.Revisions.SelectedRevision()
		if rev == nil {
			return nil, true
		}
		p.s.setFocus(PanelJournal)
		cmd, _ := p.s.core.Revisions.HandleIntent(intents.OpenInlineDescribe{Selected: rev})
		return cmd, true
	}
	return nil, false
}

// filesPanel hosts the Files model (upstream's file-details operation).
type filesPanel struct {
	s *Shell
	m *files.Model
}

func (p *filesPanel) title() string                                    { return "Files" }
func (p *filesPanel) init() tea.Cmd                                    { return nil }
func (p *filesPanel) update(msg tea.Msg) tea.Cmd                       { return p.m.Update(msg) }
func (p *filesPanel) setFocused(bool)                                  {}
func (p *filesPanel) editing() bool                                    { return p.m.IsEditing() }
func (p *filesPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *filesPanel) scopes() []common.Scope                           { return p.m.Scopes(p.s.h) }
func (p *filesPanel) selection() common.SelectionSnapshot              { return p.m.Selection() }
func (p *filesPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	return p.m.HandleIntent(intent)
}
