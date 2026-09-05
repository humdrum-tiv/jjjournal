// Package statuspanel is jjj's Status panel (spec §7): three lines refreshed
// after every load, computed off the UI loop by statusinfo.Gather.
package statuspanel

import (
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/statusinfo"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type infoMsg struct{ info statusinfo.Info }

type Model struct {
	ctx    *appContext.MainContext
	mode   func() string
	info   statusinfo.Info
	loaded bool
}

// New builds the panel; mode reports "journal" or "graph" at gather time.
func New(ctx *appContext.MainContext, mode func() string) *Model {
	return &Model{ctx: ctx, mode: mode}
}

func (m *Model) Init() tea.Cmd { return m.reload() }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case infoMsg:
		m.info, m.loaded = msg.info, true
	case common.RefreshMsg, common.UpdateRevisionsSuccessMsg:
		return m.reload()
	}
	return nil
}

func (m *Model) reload() tea.Cmd {
	// Read everything on the UI loop; the closure runs on another goroutine.
	ctx, mode := m.ctx, m.mode()
	root, revset := ctx.Location, ctx.CurrentRevset
	remote := config.GetGitDefaultRemote(config.Current)
	return func() tea.Msg {
		return infoMsg{info: statusinfo.Gather(ctx, root, remote, revset, mode, time.Now())}
	}
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	lines := m.info.Lines(time.Now())
	if !m.loaded {
		lines = [3]string{filepath.Base(m.ctx.Location), "…", m.ctx.CurrentRevset}
	}
	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase)
	for i, line := range lines {
		if i > 0 {
			tb.NewLine()
		}
		tb.Write(ansi.Truncate(line, box.R.Dx(), "…"))
	}
	tb.Done()
}
