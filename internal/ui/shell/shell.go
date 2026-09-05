package shell

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
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
	ctx     *context.MainContext
	cfg     *jjj.Config
	detail  *preview.Model
	focus   PanelID
	core    ui.Core
	hasCore bool
	// active is true only while Render last drew the panels. It gates
	// HandleKey so digits/tab/scroll keys fall through to the stock layout
	// when the shell is disabled or the terminal is too narrow to show it.
	active bool
}

func New(ctx *context.MainContext, cfg *jjj.Config) *Shell {
	return &Shell{ctx: ctx, cfg: cfg, detail: preview.New(ctx), focus: PanelJournal, active: true}
}

// Attach receives the stock models before Init; Task 5 installs the journal
// row-source factory here.
func (s *Shell) Attach(core ui.Core) { s.core, s.hasCore = core, true }

func (s *Shell) Init() tea.Cmd { return s.detail.Init() }

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
	return s.detail.Update(msg), false
}

// HandleKey: digits and tab move focus; when Detail is focused the movement
// keys scroll it. Everything else falls through to upstream. Printable keys
// are matched on press.Text (the character actually typed, so "J" and "{"
// work however the terminal reports shift); named keys on String().
func (s *Shell) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if !s.active {
		return nil, false
	}
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}
	text, key := press.Text, press.String()
	if id, ok := focusKeys[text]; ok {
		s.setFocus(id)
		return nil, true
	}
	switch key {
	case "tab":
		s.setFocus(s.focus.next())
		return nil, true
	case "shift+tab":
		s.setFocus(s.focus.prev())
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
	lay := Compute(content, s.cfg.Shell.LeftWidth)

	s.panel(dl, lay.Status, PanelStatus, "Status", func(inner layout.Box) {
		s.drawStatusPlaceholder(dl, inner)
	})
	s.panel(dl, lay.Journal, PanelJournal, "Journal", func(inner layout.Box) {
		rows := inner.V(layout.Fixed(1), layout.Fill(1))
		if len(rows) < 2 {
			return
		}
		core.Revset.ViewRect(dl, rows[0])
		core.RenderMain(rows[1])
	})
	s.panel(dl, lay.Detail, PanelDetail, "Detail", func(inner layout.Box) {
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

// drawStatusPlaceholder is replaced by the statuspanel model in Task 6.
func (s *Shell) drawStatusPlaceholder(dl *render.DisplayContext, inner layout.Box) {
	dl.Text(inner.R.Min.X, inner.R.Min.Y, render.ZBase).
		Write(filepath.Base(s.ctx.Location)).
		NewLine().
		Write(s.ctx.CurrentRevset).
		Done()
}
