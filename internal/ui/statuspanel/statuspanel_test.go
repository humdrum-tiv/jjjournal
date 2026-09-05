package statuspanel

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/statusinfo"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

func newTestModel() *Model {
	return New(&appContext.MainContext{Location: "/tmp/x", CurrentRevset: "all()"}, func() string { return "journal" })
}

// TestUpdateReloadTriggers covers F2(a): only common.UpdateRevisionsSuccessMsg
// reloads; common.RefreshMsg must not (a refresh fires before the load even
// starts, so reacting to it just re-gathers stale state and, in graph mode,
// once per batch).
func TestUpdateReloadTriggers(t *testing.T) {
	m := newTestModel()

	cmd := m.Update(common.UpdateRevisionsSuccessMsg{})
	require.NotNil(t, cmd, "UpdateRevisionsSuccessMsg must trigger a reload")

	cmd = m.Update(common.RefreshMsg{})
	require.Nil(t, cmd, "RefreshMsg must not trigger a reload")
}

// TestUpdateIgnoresStaleSeq covers F2(c): an infoMsg carrying a seq older
// than the latest issued reload must be dropped, so a slow Gather from a
// superseded reload can never clobber a newer result.
func TestUpdateIgnoresStaleSeq(t *testing.T) {
	m := newTestModel()
	m.seq = 2

	stale := statusinfo.Info{Repo: "stale"}
	cmd := m.Update(infoMsg{info: stale, seq: 1})
	require.Nil(t, cmd)
	require.False(t, m.loaded, "a stale infoMsg must not be applied")
	require.Equal(t, statusinfo.Info{}, m.info)

	current := statusinfo.Info{Repo: "current"}
	cmd = m.Update(infoMsg{info: current, seq: 2})
	require.Nil(t, cmd)
	require.True(t, m.loaded)
	require.Equal(t, current, m.info)
}

// TestViewRectPlaceholderBeforeFirstResult covers F2's placeholder path:
// before the first Gather result lands, ViewRect draws the repo basename,
// an ellipsis, and the revset rather than blocking or showing nothing.
func TestViewRectPlaceholderBeforeFirstResult(t *testing.T) {
	m := newTestModel()

	dl := render.NewDisplayContext()
	box := layout.Box{R: layout.Rect(0, 0, 40, 3)}
	m.ViewRect(dl, box)

	out := ansi.Strip(dl.RenderToString(40, 3))
	require.Contains(t, out, "x") // filepath.Base("/tmp/x")
	require.Contains(t, out, "…")
	require.Contains(t, out, "all()")
}
