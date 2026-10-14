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
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/files"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/notes"
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
	// normalMode reports whether no upstream operation, modal or quick
	// search is open on the revisions list; tests replace it.
	normalMode func() bool
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
		return s.hasCore && s.core.Revisions.InNormalMode() && !s.core.Revisions.HasQuickSearch()
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
	for _, id := range Order {
		if p, ok := s.panels[id]; ok {
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

func (s *Shell) setFocus(id PanelID) {
	if !s.present()[id] {
		return
	}
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
		s.setFocus(Cycle(s.present(), s.focus, it.Delta))
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
// every message that is neither input nor an intent to all panels. Intents
// travelling as messages must not reach hosted upstream models, which would
// act on them (a Cancel closing a confirmation, say).
func (s *Shell) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg.(type) {
	case focusPanelMsg:
		s.setFocus(msg.(focusPanelMsg).id)
		return nil, true
	case revisions.PaneClickedMsg, revisions.ItemClickedMsg:
		s.setFocus(PanelJournal) // let revisions handle the click too
		return nil, false
	case details.FileClickedMsg:
		s.setFocus(PanelFiles)
		return s.panels[PanelFiles].update(msg), true
	case files.DetailContentMsg:
		s.detail.SetContent(msg.(files.DetailContentMsg).Content)
		return nil, true
	case common.CloseViewMsg:
		// esc in Files or Bookmarks (upstream emits Close) returns to Journal.
		if s.focus == PanelFiles || s.focus == PanelBookmarks {
			s.setFocus(PanelJournal)
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
	for _, id := range Order {
		if p, ok := s.panels[id]; ok {
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
	content, statusLine := box.CutBottom(1)
	if box.R.Dx() < s.cfg.Shell.NarrowColumns {
		if s.focus == PanelJournal {
			return false
		}
		s.drawPanel(dl, content, s.focus)
		core.Status.ViewRect(dl, statusLine)
		return true
	}
	lay := Compute(content, Params{
		LeftWidth: s.cfg.Shell.LeftWidth,
		NotesRows: s.cfg.Shell.NotesRows,
		Focus:     s.focus,
		Zoom:      s.zoom,
		Present:   s.present(),
	})
	for _, id := range Order {
		if b, ok := lay[id]; ok {
			s.drawPanel(dl, b, id)
		}
	}
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
		Width(w - 2).
		Height(h - 2).
		Render("")
	dl.AddDraw(box.R, frame, render.ZBase)
	dl.AddInteraction(box.R, focusPanelMsg{id: id}, render.InteractionClick, -1)
	dl.Text(box.R.Min.X+2, box.R.Min.Y, render.ZBase).
		Styled(fmt.Sprintf("[%d]─%s", int(id), p.title()), style.Bold(s.focus == id)).
		Done()
	p.draw(dl, box.Inset(1))
}
