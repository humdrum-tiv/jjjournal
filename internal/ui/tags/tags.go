package tags

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// loadedMsg carries one Load's result; only the latest seq is applied.
type loadedMsg struct {
	seq  int
	tags []Tag
}

type Model struct {
	ctx     *appContext.MainContext
	tags    []Tag
	loaded  bool
	cursor  int
	offset  int
	seq     int
	focused bool
}

func New(ctx *appContext.MainContext) *Model { return &Model{ctx: ctx} }

// Load lists your tags: the bookmark list names the foreign remotes, whose
// tags the tag list then marks for ParseTags to drop.
func (m *Model) Load() tea.Cmd {
	m.seq++
	seq, ctx := m.seq, m.ctx
	return func() tea.Msg {
		var foreign []string
		if out, err := ctx.RunCommandImmediate(jj.BookmarkListAll()); err == nil {
			foreign = ForeignRemotes(string(out))
		}
		out, err := ctx.RunCommandImmediate(jj.TagListJournal(foreign))
		if err != nil {
			return loadedMsg{seq: seq}
		}
		return loadedMsg{seq: seq, tags: ParseTags(string(out))}
	}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if msg, ok := msg.(loadedMsg); ok && msg.seq == m.seq {
		m.tags, m.loaded = msg.tags, true
		m.cursor = min(m.cursor, max(len(m.tags)-1, 0))
	}
	return nil
}

func (m *Model) SetFocused(focused bool) { m.focused = focused }

// Selected is the tag under the cursor.
func (m *Model) Selected() (Tag, bool) {
	if m.cursor < 0 || m.cursor >= len(m.tags) {
		return Tag{}, false
	}
	return m.tags[m.cursor], true
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch it := intent.(type) {
	case intents.TagsMove:
		m.cursor = max(0, min(len(m.tags)-1, m.cursor+it.Delta))
		return nil, true
	case intents.TagsReveal:
		tag, ok := m.Selected()
		if !ok || tag.Conflict {
			return nil, true
		}
		return func() tea.Msg { return bookmarkpane.RevealRevisionMsg{CommitID: tag.CommitID} }, true
	}
	return nil, false
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	w, h := box.R.Dx(), box.R.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	p := common.DefaultPalette
	dimmed := p.Get("picker", "", "dimmed", false)
	if !m.loaded {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Styled("…", dimmed).Done()
		return
	}
	if len(m.tags) == 0 {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Styled("No tags", dimmed).Done()
		return
	}
	nameStyle := p.Get("picker", "", "bookmark", false)
	conflict := p.Get("", "", "error", false)
	text := lipgloss.NewStyle()

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	for row := range min(h, len(m.tags)-m.offset) {
		i := m.offset + row
		tag := m.tags[i]
		y := box.R.Min.Y + row
		if i == m.cursor && m.focused {
			dl.AddHighlight(layout.Rect(box.R.Min.X, y, w, 1), p.Get("revisions", "", "", true), render.ZBase+1)
		}
		tb := dl.Text(box.R.Min.X, y, render.ZBase)
		left := w
		put := func(s string, st lipgloss.Style) {
			if left <= 0 {
				return
			}
			if render.StringWidth(s) > left {
				s = ansi.Truncate(s, left, "…")
			}
			left -= render.StringWidth(s)
			tb.Styled(s, st)
		}
		put(tag.Name, nameStyle)
		if tag.Conflict {
			put("  ", text)
			put("conflict", conflict)
		} else {
			put("  ", text)
			put(tag.When.Format("2006-01-02"), dimmed)
			put("  ", text)
			put(tag.Subject, text)
		}
		tb.Done()
	}
}
