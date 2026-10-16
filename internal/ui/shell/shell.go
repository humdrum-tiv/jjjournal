package shell

import (
	gocontext "context"
	"fmt"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/journal"
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/confirmation"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/files"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/notes"
	bookmarkop "github.com/idursun/jjui/internal/ui/operations/bookmark"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
	"github.com/idursun/jjui/internal/ui/statuspanel"
)

// focusPanelMsg is produced by clicking a panel's rectangle.
type focusPanelMsg struct{ id PanelID }

var (
	borderFocused   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	borderUnfocused = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Shell is the ui.Extension that lays out jjj's panels and routes keys to
// them through the scopes it contributes (ui.go's ScopeProvider seam).
type Shell struct {
	ctx     *appContext.MainContext
	cfg     *jjj.Config
	core    ui.Core
	hasCore bool

	panels map[PanelID]panel
	focus  PanelID
	zoom   Zoom
	detail *preview.Model // also driven directly by Files (annotate)
	h      handler
	// normalMode reports whether no upstream modal, overlay, text input,
	// split-pane focus or revision operation is open (Core.InNormalMode);
	// tests replace it.
	normalMode func() bool
	// width, height: the last window size; zero until the first
	// tea.WindowSizeMsg. Focus and the Files load use the layout they give.
	width, height int
	// click is the last mouse click; upstream's click messages carry no
	// position, so it tells a click in a jjj panel from one in upstream's
	// inline details or split pane.
	click tea.Mouse
	// moveFromBookmarks: the pending bookmark move began in Bookmarks, so
	// cancelling it returns there.
	moveFromBookmarks bool
	// detailFile, while Detail is focused, keeps Detail on a file diff
	// opened from Files (enter) instead of the selected revision.
	detailFile *common.SelectedFile

	journalMode atomic.Bool                    // read from the load goroutine
	source      atomic.Pointer[journal.Source] // last journal source, for jump keys
	dayStyle    lipgloss.Style                 // header styles, resolved on the UI loop in Attach
	weekStyle   lipgloss.Style
}

var (
	_ common.ScopeProvider     = (*Shell)(nil)
	_ common.SelectionProvider = (*Shell)(nil)
)

// panelNames maps [shell] hidden entries to panels. Journal and Detail
// cannot be hidden.
var panelNames = map[string]PanelID{
	"status": PanelStatus, "files": PanelFiles, "bookmarks": PanelBookmarks, "notes": PanelNotes,
}

func New(ctx *appContext.MainContext, cfg *jjj.Config) *Shell {
	s := &Shell{ctx: ctx, cfg: cfg, panels: map[PanelID]panel{}, focus: PanelJournal, detail: preview.New(ctx)}
	s.h = handler{s}
	s.normalMode = func() bool {
		if !s.hasCore {
			return false
		}
		if s.core.InNormalMode != nil {
			return s.core.InNormalMode()
		}
		return s.core.Revisions.InNormalMode() && !s.core.Revisions.HasQuickSearch()
	}
	s.journalMode.Store(cfg.Journal.Enabled)
	status := statuspanel.New(ctx, func() string {
		if s.journalMode.Load() {
			return "journal"
		}
		return "graph"
	})
	s.register(PanelStatus, &statusPanel{m: status, h: s.h})
	s.register(PanelJournal, &journalPanel{s: s})
	s.register(PanelDetail, &detailPanel{m: s.detail, h: s.h})
	s.register(PanelNotes, &notesPanel{s: s, m: notes.New(ctx)})
	s.register(PanelFiles, &filesPanel{s: s, m: files.New(ctx, func() bool {
		if !s.hasCore {
			return false
		}
		rev := s.core.Revisions.SelectedRevision()
		return rev != nil && rev.IsWorkingCopy
	})})
	bookmarks := bookmarkpane.New(ctx)
	bookmarks.EnableJournalView(cfg.Bookmarks.Pinned)
	s.register(PanelBookmarks, &bookmarksPanel{s: s, m: bookmarks})
	return s
}

// Attach receives the stock models before Init and installs the journal
// row-source factory so the very first load already uses journal mode.
func (s *Shell) Attach(core ui.Core) {
	s.core, s.hasCore = core, true
	dim := common.DefaultPalette.Get("revisions", "", "dimmed", false)
	s.dayStyle, s.weekStyle = dim.Bold(true), dim
	// Journal mode is delivered entirely through the streaming row source
	// (rowSource below): revisions.Model.refresh only consults the factory
	// on the streaming path and calls jj.Log directly otherwise, so a
	// user's carried-over `log_batching = false` (e.g. from a jjui config)
	// would silently bypass journal mode. Force it on.
	config.Current.Revisions.LogBatching = true
	core.Revisions.SetRowSourceFactory(s.rowSource)
}

// rowSource is the revisions.RowSourceFactory: the journal source in journal
// mode, upstream's graph streamer otherwise. It runs inside the list's load
// command (a goroutine), hence the atomics and the styles pre-resolved in Attach.
func (s *Shell) rowSource(ctx gocontext.Context, runner appContext.CommandRunner, revset, jjTemplate string) (revisions.RowSource, error) {
	if !s.journalMode.Load() {
		s.source.Store(nil)
		return revisions.DefaultRowSource(ctx, runner, revset, jjTemplate)
	}
	tmpl := s.cfg.Journal.Template
	if tmpl == "" {
		tmpl = journal.RowTemplateJournal
	}
	src, err := journal.NewSource(runner, revset, tmpl, s.cfg.Journal.Limit, journal.Options{
		Now:       time.Now(),
		DayStyle:  s.dayStyle,
		WeekStyle: s.weekStyle,
	})
	if err != nil {
		return nil, err // untyped nil, see revisions.DefaultRowSource
	}
	s.source.Store(src)
	return src, nil
}

func (s *Shell) register(id PanelID, p panel) { s.panels[id] = p }

func (s *Shell) Init() tea.Cmd {
	var cmds []tea.Cmd
	present := s.present()
	for _, id := range Order {
		if p, ok := s.panels[id]; ok && present[id] {
			cmds = append(cmds, p.init())
		}
	}
	return tea.Batch(cmds...)
}

// present is the set of registered, non-hidden panels.
func (s *Shell) present() map[PanelID]bool {
	out := make(map[PanelID]bool, len(s.panels))
	for id := range s.panels {
		out[id] = true
	}
	for _, name := range s.cfg.Shell.Hidden {
		if id, ok := panelNames[name]; ok {
			delete(out, id)
		}
	}
	return out
}

// setFocus moves focus to a focusable panel. It refuses (false) while the
// focused panel is editing, so a confirmation or filter is never left open
// behind another panel where a later click could fire it.
func (s *Shell) setFocus(id PanelID) bool {
	if !s.focusable()[id] {
		return false
	}
	if p := s.panels[s.focus]; id != s.focus && p != nil && p.editing() {
		return false
	}
	s.moveFocus(id)
	return true
}

// moveFocus moves focus unconditionally.
func (s *Shell) moveFocus(id PanelID) {
	if id != s.focus {
		s.zoom = ZoomNone
	}
	if id != PanelDetail {
		s.detailFile = nil
	}
	s.focus = id
	for pid, p := range s.panels {
		p.setFocused(pid == id)
	}
}

// drawable mirrors drawPanel's minimum outer size.
func drawable(b layout.Box) bool { return b.R.Dx() >= 4 && b.R.Dy() >= 3 }

// arrange is the layout for a full-window box: the panel boxes, and whether
// the shell draws at all (narrow mode with Journal focused draws the stock
// layout). Render, focus and the Files load all use it.
func (s *Shell) arrange(box layout.Box, focus PanelID, zoom Zoom) (Layout, bool) {
	content, _ := box.CutBottom(1)
	if box.R.Dx() < s.cfg.Shell.NarrowColumns {
		if focus == PanelJournal {
			return nil, false
		}
		return Layout{focus: content}, true
	}
	return Compute(content, Params{
		LeftWidth: s.cfg.Shell.LeftWidth,
		NotesRows: s.cfg.Shell.NotesRows,
		Focus:     focus,
		Zoom:      zoom,
		Present:   s.present(),
	}), true
}

func (s *Shell) window() (layout.Box, bool) {
	if s.width <= 0 || s.height <= 0 {
		return layout.Box{}, false
	}
	return layout.NewBox(layout.Rect(0, 0, s.width, s.height)), true
}

// focusable: present panels that would get a drawable box if focused
// (Notes with notes_rows = 0 or a short right column, cramped accordion
// panels do not). Journal always qualifies; before the first window size
// every present panel does.
func (s *Shell) focusable() map[PanelID]bool {
	present := s.present()
	win, ok := s.window()
	if !ok {
		return present
	}
	out := map[PanelID]bool{PanelJournal: true}
	for id := range present {
		lay, _ := s.arrange(win, id, ZoomNone)
		if b, ok := lay[id]; ok && drawable(b) {
			out[id] = true
		}
	}
	return out
}

// drawn: the panels with a drawable box this frame; every present panel
// before the first window size.
func (s *Shell) drawn() map[PanelID]bool {
	win, ok := s.window()
	if !ok {
		return s.present()
	}
	lay, _ := s.arrange(win, s.focus, s.zoom)
	out := map[PanelID]bool{}
	for id, b := range lay {
		if drawable(b) {
			out[id] = true
		}
	}
	return out
}

// syncVisibility tells Files whether it is drawn; a load deferred while it
// was not runs when it becomes drawn.
func (s *Shell) syncVisibility() tea.Cmd {
	fp, ok := s.panels[PanelFiles].(*filesPanel)
	if !ok {
		return nil
	}
	return fp.m.SetVisible(s.drawn()[PanelFiles])
}

// clickIn reports whether the last click fell inside panel id's box.
func (s *Shell) clickIn(id PanelID) bool {
	win, ok := s.window()
	if !ok {
		return false
	}
	lay, shown := s.arrange(win, s.focus, s.zoom)
	b, inLay := lay[id]
	return shown && inLay && drawable(b) && s.click.X >= b.R.Min.X && s.click.X < b.R.Max.X &&
		s.click.Y >= b.R.Min.Y && s.click.Y < b.R.Max.Y
}

// Scopes: the focused panel's scopes (innermost first), then jjj.shell,
// which is LeakAll (revisions stay reachable while Journal is focused) and
// Global (still visible behind a LeakGlobal panel scope). Nothing while an
// upstream operation is open or a panel owns raw input — except that panel's
// own scopes.
func (s *Shell) Scopes() []common.Scope {
	if !s.cfg.Shell.Enabled || !s.normalMode() {
		return nil
	}
	p := s.panels[s.focus]
	var scopes []common.Scope
	if p != nil {
		scopes = append(scopes, p.scopes()...)
		if p.editing() {
			return scopes
		}
	}
	return append(scopes, common.Scope{Name: actions.ScopeJjjShell, Leak: common.LeakAll, Global: true, Handler: s.h})
}

// Selection: a file opened into Detail, else the focused panel's selection.
func (s *Shell) Selection() common.SelectionSnapshot {
	if s.focus == PanelDetail && s.detailFile != nil {
		return common.SelectionSnapshot{Highlighted: *s.detailFile}
	}
	if sp, ok := s.panels[s.focus].(selectionPanel); ok {
		return sp.selection()
	}
	return common.SelectionSnapshot{}
}

func (s *Shell) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch it := intent.(type) {
	case intents.ShellFocusPanel:
		s.setFocus(PanelID(it.Panel))
		return nil, true
	case intents.ShellCyclePanel:
		s.setFocus(Cycle(s.focusable(), s.focus, it.Delta))
		return nil, true
	case intents.FocusNextPane:
		// upstream's bookmark_pane binds tab to ui.focus_next_pane, which
		// routes through jjj.shell before ui's no-op handler.
		s.setFocus(Cycle(s.focusable(), s.focus, 1))
		return nil, true
	case intents.ShellZoom:
		s.zoom = s.zoom.Step(it.Delta)
		return nil, true
	case intents.ShellToggleJournal:
		s.journalMode.Store(!s.journalMode.Load())
		s.source.Store(nil) // drop the stale source: {/} must not use old header indices during the reload
		return common.RefreshAndKeepSelections, true
	case intents.ShellJumpDay:
		if s.focus != PanelJournal {
			return nil, false
		}
		if src := s.source.Load(); src != nil && s.hasCore {
			if idx := journal.JumpTarget(src.HeaderIndices(), s.core.Revisions.Cursor(), it.Forward); idx >= 0 {
				s.core.Revisions.SetCursor(idx)
			}
		}
		return nil, true
	case intents.FilesShowDiff:
		fp, ok := s.panels[PanelFiles].(*filesPanel)
		if !ok {
			return nil, true
		}
		if f, ok := fp.m.Current(); ok {
			s.setFocus(PanelDetail)
			s.detailFile = &f
		}
		return nil, true
	}
	if ip, ok := s.panels[s.focus].(intentPanel); ok {
		return ip.handleIntent(intent)
	}
	return nil, false
}

// Update handles panel clicks and focus-returning closes, then forwards
// every message that is neither input nor an intent to the present panels.
// Intents travelling as messages must not reach hosted upstream models,
// which would act on them (a Cancel closing a confirmation, say).
func (s *Shell) Update(msg tea.Msg) (tea.Cmd, bool) {
	cmd, handled := s.update(msg)
	return tea.Batch(cmd, s.syncVisibility()), handled
}

func (s *Shell) update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width, s.height = msg.Width, msg.Height
		if !s.focusable()[s.focus] {
			s.moveFocus(PanelJournal)
		}
	case tea.MouseClickMsg:
		s.click = msg.Mouse()
		return nil, false
	case focusPanelMsg:
		s.setFocus(msg.id)
		return nil, true
	case revisions.PaneClickedMsg, revisions.ItemClickedMsg:
		if !s.setFocus(PanelJournal) {
			return nil, true // a panel is editing: the click must not move the cursor under it
		}
		return nil, false // let revisions handle the click too
	case intents.OpenSquash:
		// upstream file-details (hosted in Files) starts a squash on the
		// revisions list; follow it there so its keys and focus line up.
		s.setFocus(PanelJournal)
		return nil, false
	case details.FileClickedMsg:
		// only a click inside the Files panel; one in upstream's inline
		// details (l) indexes a different list and stays upstream's.
		if !s.normalMode() || !s.clickIn(PanelFiles) {
			return nil, false
		}
		if !s.setFocus(PanelFiles) {
			return nil, true
		}
		return s.panels[PanelFiles].update(msg), true
	case files.DetailContentMsg:
		s.detail.SetContent(msg.Content)
		return nil, true
	case bookmarkpane.PaneClickedMsg, bookmarkpane.ItemClickedMsg, bookmarkpane.RemoteClickedMsg:
		// likewise, a click in upstream's split bookmark pane (b) stays upstream's.
		if !s.normalMode() || !s.clickIn(PanelBookmarks) {
			return nil, false
		}
		if !s.setFocus(PanelBookmarks) {
			return nil, true
		}
		return s.panels[PanelBookmarks].update(msg), true
	case bookmarkpane.BeginMoveBookmarkMsg:
		s.moveFromBookmarks = s.focus == PanelBookmarks
		s.setFocus(PanelJournal) // upstream starts the move on the revisions list
		return nil, false
	case bookmarkop.MoveBookmarkCancelledMsg:
		if s.moveFromBookmarks {
			s.setFocus(PanelBookmarks)
		}
		s.moveFromBookmarks = false
		return nil, false
	case bookmarkpane.RevealRevisionMsg, bookmarkpane.BeginCreateBookmarkMsg:
		// upstream (ui.handleSplitMsg) reveals the revision or starts the
		// create operation on the revisions list; follow it there.
		s.setFocus(PanelJournal)
		return nil, false
	case confirmation.SelectOptionMsg, confirmation.CloseMsg:
		// a confirmation belongs to the focused panel (focus cannot leave a
		// panel while it is editing); never fire one left in another.
		if p, ok := s.panels[s.focus]; ok && s.present()[s.focus] {
			return p.update(msg), false
		}
		return nil, false
	case common.CloseViewMsg:
		// esc in Files or Bookmarks (upstream emits Close) returns to Journal.
		// Upstream's split "Yes"/"Interactive" and restore "Interactive" also
		// close with common.Close rather than confirmation.Close: a hosted
		// model survives the close, so clear its confirmation first.
		if s.focus == PanelFiles || s.focus == PanelBookmarks {
			cmd := s.panels[s.focus].update(confirmation.CloseMsg{})
			s.setFocus(PanelJournal)
			return cmd, false
		}
		return nil, false
	}
	if common.IsInputMessage(msg) {
		return nil, false
	}
	if _, ok := msg.(intents.Intent); ok {
		return nil, false
	}
	var cmds []tea.Cmd
	present := s.present()
	for _, id := range Order {
		if p, ok := s.panels[id]; ok && present[id] {
			cmds = append(cmds, p.update(msg))
		}
	}
	return tea.Batch(cmds...), false
}

// HandleKey is inert: every jjj key is an action in a jjj scope (Scopes),
// so it is rebindable and routed by upstream's dispatcher.
func (s *Shell) HandleKey(tea.KeyMsg) (tea.Cmd, bool) { return nil, false }

// Render draws the panels. Below [shell] narrow_columns it draws the stock
// layout while Journal is focused, and the focused panel full-screen
// otherwise (digits still focus panels there).
func (s *Shell) Render(dl *render.DisplayContext, box layout.Box, core ui.Core) bool {
	s.core, s.hasCore = core, true
	if !s.cfg.Shell.Enabled {
		return false
	}
	lay, shown := s.arrange(box, s.focus, s.zoom)
	if !shown {
		return false
	}
	for _, id := range Order {
		if b, ok := lay[id]; ok {
			s.drawPanel(dl, b, id)
		}
	}
	_, statusLine := box.CutBottom(1)
	core.Status.ViewRect(dl, statusLine)
	return true
}

// drawPanel draws a rounded border with a "[n]─Title" tab, registers a click
// target that focuses the panel, then draws the content inside the border.
func (s *Shell) drawPanel(dl *render.DisplayContext, box layout.Box, id PanelID) {
	p := s.panels[id]
	w, h := box.R.Dx(), box.R.Dy()
	if p == nil || w < 4 || h < 3 {
		return
	}
	style := borderUnfocused
	if s.focus == id {
		style = borderFocused
	}
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.GetForeground()).
		Width(w).
		Height(h).
		Render("")
	dl.AddDraw(box.R, frame, render.ZBase)
	dl.AddInteraction(box.R, focusPanelMsg{id: id}, render.InteractionClick, -1)
	dl.Text(box.R.Min.X+2, box.R.Min.Y, render.ZBase).
		Styled(fmt.Sprintf("[%d]─%s", int(id), p.title()), style.Bold(s.focus == id)).
		Done()
	p.draw(dl, box.Inset(1))
}
