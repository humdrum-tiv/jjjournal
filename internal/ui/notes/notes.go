// Package notes is jjj's Notes panel (spec §4.4): the selected revision's
// full description, wrapped, first line bold.
package notes

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	debounceID       = "jjj.notes"
	debounceDuration = 50 * time.Millisecond
)

type descMsg struct {
	changeID, text string
	seq            uint64
	err            error
}

type Model struct {
	ctx      *appContext.MainContext
	changeID string
	text     string
	loaded   bool
	err      error
	seq      uint64
	offset   int
}

func New(ctx *appContext.MainContext) *Model { return &Model{ctx: ctx} }

func (m *Model) ChangeID() string { return m.changeID }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		rev, ok := msg.Item.(common.SelectedRevision)
		if !ok || rev.ChangeId == m.changeID {
			return nil
		}
		m.changeID, m.offset, m.loaded = rev.ChangeId, 0, false
		return m.reload()
	case common.UpdateRevisionsSuccessMsg:
		if m.changeID != "" {
			return m.reload()
		}
	case descMsg:
		if msg.seq == m.seq && msg.changeID == m.changeID {
			m.text, m.err, m.loaded = msg.text, msg.err, true
		}
	}
	return nil
}

func (m *Model) reload() tea.Cmd {
	m.seq++
	seq, id, ctx := m.seq, m.changeID, m.ctx
	return common.Debounce(debounceID, debounceDuration, func() tea.Msg {
		out, err := ctx.RunCommandImmediate(jj.GetDescription(id))
		if err != nil {
			return descMsg{changeID: id, seq: seq, err: err}
		}
		return descMsg{changeID: id, text: string(out), seq: seq}
	})
}

// Lines is the description wrapped to width; "(no description)" when empty,
// or a single "(description unavailable: ...)" line when the load failed.
func (m *Model) Lines(width int) []string {
	if m.err != nil {
		return []string{"(description unavailable: " + m.err.Error() + ")"}
	}
	text := strings.TrimRight(m.text, "\n ")
	if strings.TrimSpace(text) == "" {
		return []string{"(no description)"}
	}
	return strings.Split(ansi.Wrap(text, max(width, 1), ""), "\n")
}

func (m *Model) Scroll(delta int) { m.offset = max(0, m.offset+delta) }

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if !m.loaded {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Write("…").Done()
		return
	}
	lines := m.Lines(box.R.Dx())
	m.offset = min(m.offset, max(0, len(lines)-1))
	bold := lipgloss.NewStyle().Bold(true)
	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase)
	for i, line := range lines[m.offset:] {
		if i >= box.R.Dy() {
			break
		}
		if i > 0 {
			tb.NewLine()
		}
		if i+m.offset == 0 {
			tb.Styled(line, bold)
		} else {
			tb.Write(line)
		}
	}
	tb.Done()
}
