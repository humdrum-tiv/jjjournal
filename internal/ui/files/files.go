// Package files is jjj's Files panel (spec §4.2): the files changed in the
// Journal-selected revision, hosted from upstream's file-details operation
// so its keys (space r s S A d / *) work unchanged, plus jjj file actions.
package files

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	debounceID = "jjj.files"
	// debounceDuration: details.Operation loads synchronously (jj snapshot +
	// status) on the UI loop, so wait for the cursor to settle.
	debounceDuration = 120 * time.Millisecond
)

// revisionMsg is the debounced "load this revision" request.
type revisionMsg struct{ rev common.SelectedRevision }

// showMsg runs a load deferred while the panel was not drawn.
type showMsg struct{}

// DetailContentMsg asks the shell to show Content in Detail (annotate).
type DetailContentMsg struct{ Content string }

type Model struct {
	ctx           *appContext.MainContext
	isWorkingCopy func() bool // is the Journal-selected revision @?
	op            *details.Operation
	rev           common.SelectedRevision // latest requested
	// hidden: the panel is not drawn (narrow mode, zoom). Loads are
	// synchronous jj calls on the UI loop, so they wait (dirty) until the
	// panel is drawn again; see SetVisible.
	hidden bool
	dirty  bool
}

func New(ctx *appContext.MainContext, isWorkingCopy func() bool) *Model {
	return &Model{ctx: ctx, isWorkingCopy: isWorkingCopy}
}

// SetVisible reports whether the panel is drawn. Becoming visible with a
// deferred load pending returns the command that runs it, once.
func (m *Model) SetVisible(visible bool) tea.Cmd {
	m.hidden = !visible
	if !visible || !m.dirty {
		return nil
	}
	m.dirty = false
	return func() tea.Msg { return showMsg{} }
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		rev, ok := msg.Item.(common.SelectedRevision)
		if !ok || rev == m.rev {
			return nil
		}
		m.rev = rev
		return common.Debounce(debounceID, debounceDuration, func() tea.Msg { return revisionMsg{rev: rev} })
	case revisionMsg:
		if msg.rev != m.rev {
			return nil
		}
		if m.hidden {
			m.dirty = true
			return nil
		}
		return m.load()
	case showMsg:
		if m.hidden {
			m.dirty = true
			return nil
		}
		if m.op != nil && m.op.Current != nil && m.op.Current.CommitId == m.rev.CommitId && m.op.Current.ChangeId == m.rev.ChangeId {
			return m.op.Update(common.RefreshMsg{})
		}
		return m.load()
	case common.RefreshMsg:
		if m.op != nil && m.hidden {
			m.dirty = true
			return nil
		}
	}
	if m.op == nil {
		return nil
	}
	return m.op.Update(msg)
}

// load shows the latest requested revision (a no-op before any selection).
func (m *Model) load() tea.Cmd {
	if m.rev == (common.SelectedRevision{}) {
		return nil
	}
	commit := &jj.Commit{ChangeId: m.rev.ChangeId, CommitId: m.rev.CommitId}
	if m.op == nil {
		m.op = details.NewOperation(m.ctx, commit)
		return m.op.Init()
	}
	return m.op.Update(common.SelectionChangedMsg{Item: m.rev})
}

// Scopes: jjj.files first (LeakAll, so the details scopes behind it stay
// reachable), then upstream's details scopes (LeakGlobal: revisions are
// blocked). While the op owns raw input only its scopes are returned.
func (m *Model) Scopes(h common.ScopeHandler) []common.Scope {
	if m.op == nil {
		return []common.Scope{{Name: actions.ScopeJjjFiles, Leak: common.LeakGlobal, Handler: h}}
	}
	if m.op.IsEditing() {
		return m.op.Scopes()
	}
	return append([]common.Scope{{Name: actions.ScopeJjjFiles, Leak: common.LeakAll, Handler: h}}, m.op.Scopes()...)
}

func (m *Model) IsEditing() bool { return m.op != nil && m.op.IsEditing() }

// Selection reports only the highlighted file; checked files stay private
// so they never mix with checked revisions in the shared context.
func (m *Model) Selection() common.SelectionSnapshot {
	if m.op == nil {
		return common.SelectionSnapshot{}
	}
	return common.SelectionSnapshot{Highlighted: m.op.Selection().Highlighted}
}

func (m *Model) Current() (common.SelectedFile, bool) {
	if m.op == nil {
		return common.SelectedFile{}, false
	}
	f, ok := m.op.Selection().Highlighted.(common.SelectedFile)
	return f, ok
}

// targets: the checked files, or the highlighted one when none is checked.
func (m *Model) targets() (string, []jj.FileName) {
	cur, ok := m.Current()
	if !ok {
		return "", nil
	}
	var files []jj.FileName
	for _, c := range m.op.Selection().Checked {
		if f, ok := c.(common.SelectedFile); ok {
			files = append(files, f.File)
		}
	}
	if len(files) == 0 {
		files = []jj.FileName{cur.File}
	}
	return cur.ChangeId, files
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.FilesDiffEdit:
		rev, files := m.targets()
		if rev == "" {
			return nil, true
		}
		return m.ctx.RunInteractiveCommand(jj.DiffEditFiles(rev, files), common.Refresh), true
	case intents.FilesResolve:
		rev, files := m.targets()
		if rev == "" {
			return nil, true
		}
		return m.ctx.RunInteractiveCommand(jj.ResolveFiles(rev, files), common.Refresh), true
	case intents.FilesAnnotate:
		cur, ok := m.Current()
		if !ok {
			return nil, true
		}
		ctx := m.ctx
		return func() tea.Msg {
			out, err := ctx.RunCommandImmediate(jj.FileAnnotate(cur.ChangeId, cur.File))
			if err != nil {
				return common.CommandCompletedMsg{Output: string(out), Err: err}
			}
			return DetailContentMsg{Content: string(out)}
		}, true
	case intents.FilesOpenEditor:
		cur, ok := m.Current()
		if !ok {
			return nil, true
		}
		if !m.isWorkingCopy() {
			return intents.Invoke(intents.AddMessage{Text: "open in $EDITOR works on the working copy (@) only"}), true
		}
		line := "${EDITOR:-vi} " + cur.File.ShellEscaped()
		return func() tea.Msg { return common.ExecMsg{Line: line, Mode: common.ExecShell} }, true
	}
	return nil, false
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if m.op == nil {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Write("…").Done()
		return
	}
	m.op.ViewRect(dl, box)
}
