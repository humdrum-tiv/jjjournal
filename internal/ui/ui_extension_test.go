package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
)

// fakeExt is a minimal Extension that also provides scopes and a selection.
type fakeExt struct {
	scopes    []common.Scope
	selection common.SelectionSnapshot
}

func (f *fakeExt) Attach(Core)                                          {}
func (f *fakeExt) Init() tea.Cmd                                        { return nil }
func (f *fakeExt) Update(tea.Msg) (tea.Cmd, bool)                       { return nil, false }
func (f *fakeExt) HandleKey(tea.KeyMsg) (tea.Cmd, bool)                 { return nil, false }
func (f *fakeExt) Render(*render.DisplayContext, layout.Box, Core) bool { return false }
func (f *fakeExt) Scopes() []common.Scope                               { return f.scopes }
func (f *fakeExt) Selection() common.SelectionSnapshot                  { return f.selection }

type fakeHandler struct{}

func (fakeHandler) HandleIntent(intents.Intent) (tea.Cmd, bool) { return nil, false }
func (fakeHandler) Update(tea.Msg) tea.Cmd                      { return nil }

func newExtModel(t *testing.T, ext *fakeExt) *Model {
	t.Helper()
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	m := NewUI(ctx)
	m.ext = ext
	return m
}

func TestExtensionScopesPrecedeRevisions(t *testing.T) {
	ext := &fakeExt{scopes: []common.Scope{{Name: "jjj.fake", Leak: common.LeakAll, Handler: fakeHandler{}}}}
	m := newExtModel(t, ext)

	names := scopeNames(m.dispatchScopes())

	fake := indexOf(names, "jjj.fake")
	revs := indexOf(names, "revisions")
	require.GreaterOrEqual(t, fake, 0, "extension scope present: %v", names)
	require.Greater(t, revs, fake, "extension scope before revisions: %v", names)
}

func TestExtensionScopesAbsentWhileStackedViewOpen(t *testing.T) {
	ext := &fakeExt{scopes: []common.Scope{{Name: "jjj.fake", Leak: common.LeakAll, Handler: fakeHandler{}}}}
	m := newExtModel(t, ext)
	m.stacked = input.NewWithTitle("Prompt", "Text: ", "")

	assert.Equal(t, -1, indexOf(scopeNames(m.dispatchScopes()), "jjj.fake"))
}

func TestExtensionSelectionWinsOverRevisions(t *testing.T) {
	file := common.SelectedFile{ChangeId: "abc", CommitId: "123", File: jj.NewFileName("a.go")}
	m := newExtModel(t, &fakeExt{selection: common.SelectionSnapshot{Highlighted: file}})

	assert.Equal(t, file, m.selectionSnapshot().Highlighted)
}

func TestExtensionEmptySelectionFallsThrough(t *testing.T) {
	m := newExtModel(t, &fakeExt{})
	// No revisions loaded either: the snapshot is simply empty, not a panic.
	assert.Nil(t, m.selectionSnapshot().Highlighted)
}

func scopeNames(scopes []common.Scope) []keybindings.ScopeName {
	out := make([]keybindings.ScopeName, len(scopes))
	for i, s := range scopes {
		out[i] = s.Name
	}
	return out
}

func indexOf(names []keybindings.ScopeName, name keybindings.ScopeName) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
