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

type infoMsg struct {
	info statusinfo.Info
	seq  uint64
}

const (
	// debounceId is the identifier the reload command debounces under.
	debounceId = "statuspanel"
	// debounceDuration matches preview.go's debounceDuration: a burst of
	// 50-row batches in graph mode should run Gather once, not once per batch.
	debounceDuration = 50 * time.Millisecond
)

type Model struct {
	ctx    *appContext.MainContext
	mode   func() string
	info   statusinfo.Info
	loaded bool
	// seq is bumped on every reload and stamped into the resulting infoMsg,
	// so a slow, superseded Gather can never overwrite a newer result (they
	// run concurrently on their own goroutines with no ordering guarantee).
	seq uint64
}

// New builds the panel; mode reports "journal" or "graph" at gather time.
func New(ctx *appContext.MainContext, mode func() string) *Model {
	return &Model{ctx: ctx, mode: mode}
}

func (m *Model) Init() tea.Cmd { return m.reload() }

// Update reloads on Init (above) and on common.UpdateRevisionsSuccessMsg
// only: a success always follows a refresh, and on failure keeping the
// previously gathered info displayed is correct. common.RefreshMsg fires
// before the load even starts, so reloading on it just re-ran Gather against
// stale state and, in graph mode, once per batch.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case infoMsg:
		if msg.seq < m.seq {
			return nil // a reload issued before the latest one; ignore it
		}
		m.info, m.loaded = msg.info, true
	case common.UpdateRevisionsSuccessMsg:
		return m.reload()
	}
	return nil
}

func (m *Model) reload() tea.Cmd {
	// Read everything on the UI loop; the closure runs on another goroutine.
	m.seq++
	seq := m.seq
	ctx, mode := m.ctx, m.mode()
	root, revset := ctx.Location, ctx.CurrentRevset
	remote := config.GetGitDefaultRemote(config.Current)
	return common.Debounce(debounceId, debounceDuration, func() tea.Msg {
		return infoMsg{info: statusinfo.Gather(ctx, root, remote, revset, mode, time.Now()), seq: seq}
	})
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
