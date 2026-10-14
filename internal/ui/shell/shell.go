package shell

import (
	gocontext "context"
	"fmt"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/journal"
	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
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
	focusKeys       = map[string]PanelID{"0": PanelDetail, "1": PanelStatus, "4": PanelJournal}
)

// Shell is the ui.Extension that lays out panels 1 (Status), 4 (Journal)
// and 0 (Detail) and routes keys by focused panel.
type Shell struct {
	ctx     *appContext.MainContext
	cfg     *jjj.Config
	detail  *preview.Model
	focus   PanelID
	core    ui.Core
	hasCore bool
	status  *statuspanel.Model
	// active is true only while Render last drew the panels. It gates
	// HandleKey so digits/tab/scroll keys fall through to the stock layout
	// when the shell is disabled or the terminal is too narrow to show it.
	active bool

	journalMode atomic.Bool                    // read from the load goroutine
	source      atomic.Pointer[journal.Source] // last journal source, for jump keys
	dayStyle    lipgloss.Style                 // header styles, resolved on the UI loop in Attach
	weekStyle   lipgloss.Style
}

func New(ctx *appContext.MainContext, cfg *jjj.Config) *Shell {
	s := &Shell{ctx: ctx, cfg: cfg, detail: preview.New(ctx), focus: PanelJournal, active: true}
	s.journalMode.Store(cfg.Journal.Enabled)
	s.status = statuspanel.New(ctx, func() string {
		if s.journalMode.Load() {
			return "journal"
		}
		return "graph"
	})
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

func (s *Shell) Init() tea.Cmd { return tea.Batch(s.detail.Init(), s.status.Init()) }

// Update forwards every non-input message to our panels (the Detail preview
// listens for SelectionChangedMsg/RefreshMsg and its own content message) and
// handles panel clicks.
func (s *Shell) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case focusPanelMsg:
		if !s.active {
			return nil, true
		}
		s.setFocus(msg.id)
		return nil, true
	case revisions.PaneClickedMsg, revisions.ItemClickedMsg:
		s.setFocus(PanelJournal) // let revisions handle the click too
		return nil, false
	}
	if common.IsInputMessage(msg) {
		return nil, false
	}
	return tea.Batch(s.detail.Update(msg), s.status.Update(msg)), false
}

// HandleKey: digits and tab move focus; when Detail is focused the movement
// keys scroll it. Everything else falls through to upstream. Printable keys
// are matched on press.Text (the character actually typed, so "J" and "{"
// work however the terminal reports shift); named keys on String().
func (s *Shell) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}
	text, key := press.Text, press.String()

	// J and {/} act on the revisions list, which is on screen in both the
	// shell layout and the stock single-list layout it falls back to, so
	// they must work even while the shell itself is inactive (narrow
	// terminal). {/} additionally require PanelJournal focus, the only
	// focus reachable while inactive since setFocus is gated below.
	switch text {
	case "J":
		s.journalMode.Store(!s.journalMode.Load())
		s.source.Store(nil) // drop the stale source: {/} must not use old header indices during the reload
		return common.RefreshAndKeepSelections, true
	case "{", "}":
		src := s.source.Load()
		if src == nil || !s.hasCore || s.focus != PanelJournal {
			return nil, false
		}
		if idx := journal.JumpTarget(src.HeaderIndices(), s.core.Revisions.Cursor(), text == "}"); idx >= 0 {
			s.core.Revisions.SetCursor(idx)
		}
		return nil, true
	}

	if !s.active {
		return nil, false
	}
	if id, ok := focusKeys[text]; ok {
		s.setFocus(id)
		return nil, true
	}
	switch key {
	case "tab":
		s.setFocus(Cycle(s.present(), s.focus, 1))
		return nil, true
	case "shift+tab":
		s.setFocus(Cycle(s.present(), s.focus, -1))
		return nil, true
	}
	if s.focus == PanelDetail {
		switch {
		case text == "j" || key == "down":
			return s.detail.Scroll(1), true
		case text == "k" || key == "up":
			return s.detail.Scroll(-1), true
		case key == "ctrl+d":
			return s.detail.HalfPageDown(), true
		case key == "ctrl+u":
			return s.detail.HalfPageUp(), true
		case key == "pgdown":
			return s.detail.PageDown(), true
		case key == "pgup":
			return s.detail.PageUp(), true
		}
	}
	return nil, false
}

func (s *Shell) setFocus(id PanelID) {
	s.focus = id
	if s.hasCore {
		s.core.Revisions.SetFocused(id == PanelJournal)
	}
}

// present is the set of panels this build draws (Task 4 replaces this with
// the panel registry).
func (s *Shell) present() map[PanelID]bool {
	return map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
}

// Render draws the panels. It returns false (stock layout) when the shell is
// disabled or the terminal is narrower than [shell] narrow_columns.
func (s *Shell) Render(dl *render.DisplayContext, box layout.Box, core ui.Core) bool {
	s.core, s.hasCore = core, true
	if !s.cfg.Shell.Enabled || box.R.Dx() < s.cfg.Shell.NarrowColumns {
		s.active = false
		return false
	}
	s.active = true
	content, statusLine := box.CutBottom(1)
	lay := Compute(content, Params{
		LeftWidth: s.cfg.Shell.LeftWidth,
		NotesRows: s.cfg.Shell.NotesRows,
		Focus:     s.focus,
		Present:   s.present(),
	})

	s.panel(dl, lay[PanelStatus], PanelStatus, "Status", func(inner layout.Box) {
		s.status.ViewRect(dl, inner)
	})
	s.panel(dl, lay[PanelJournal], PanelJournal, "Journal", func(inner layout.Box) {
		rows := inner.V(layout.Fixed(1), layout.Fill(1))
		if len(rows) < 2 {
			return
		}
		core.Revset.ViewRect(dl, rows[0])
		core.RenderMain(rows[1])
	})
	s.panel(dl, lay[PanelDetail], PanelDetail, "Detail", func(inner layout.Box) {
		s.detail.ViewRect(dl, inner)
	})
	core.Status.ViewRect(dl, statusLine)
	return true
}

// panel draws a rounded border with a "[n]─Title" tab, registers a click
// target that focuses the panel, then draws the content inside the border.
func (s *Shell) panel(dl *render.DisplayContext, box layout.Box, id PanelID, title string, draw func(inner layout.Box)) {
	w, h := box.R.Dx(), box.R.Dy()
	if w < 4 || h < 3 {
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
		Styled(fmt.Sprintf("[%d]─%s", int(id), title), style.Bold(s.focus == id)).
		Done()
	draw(box.Inset(1))
}
