package shell

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

func TestFocusTagsShowsTagsTab(t *testing.T) {
	s := newTestShell()
	_, handled := s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	require.True(t, handled)
	require.Equal(t, PanelBookmarks, s.focus)
	got := names(s.Scopes())
	require.Equal(t, actions.ScopeJjjTags, string(got[0]))
	require.Equal(t, actions.ScopeJjjShell, string(got[len(got)-1]))
}

func TestFocusRemotesSwitchesBack(t *testing.T) {
	s := newTestShell()
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelBookmarks)})
	got := names(s.Scopes())
	require.Equal(t, actions.ScopeJjjBookmarks, string(got[0]))
}

func TestTagsIntentsReachTagsModel(t *testing.T) {
	s := newTestShell()
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	_, handled := s.handleIntent(intents.TagsMove{Delta: 1})
	require.True(t, handled)
}

func TestTagsKeyRefusedWhileRemotesEditing(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	bp := s.panels[PanelBookmarks].(*bookmarksPanel)
	bp.m.HandleIntent(intents.BookmarkPaneOpenFilter{})
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	require.Equal(t, tabRemotes, bp.tab)
}

func TestRevealUnloadedTagStaysInTags(t *testing.T) {
	s := newTestShell()
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	_, handled := s.Update(bookmarkpane.RevealRevisionMsg{CommitID: "not-loaded"})
	require.False(t, handled, "upstream still sees the reveal")
	require.Equal(t, PanelBookmarks, s.focus, "nothing to show in Journal: stay")
}

func TestHiddenTagsDisablesTab(t *testing.T) {
	s := newTestShell("tags")
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelTags)})
	require.NotEqual(t, PanelBookmarks, s.focus)
	require.Equal(t, actions.ScopeJjjShell, string(names(s.Scopes())[0]), "no tags scope")
}

func TestGuideHasTagsGroup(t *testing.T) {
	groups := Guide([]config.BindingConfig{
		{Action: "jjj.tags.reveal", Scope: "jjj.tags", Key: config.StringList{"enter"}},
	})
	var names []string
	for _, g := range groups {
		names = append(names, g.Name)
	}
	require.Contains(t, names, "Common tasks · Remotes (3)")
	require.Contains(t, names, "Common tasks · Tags (8)")
}

func TestPanelThreeTabTitle(t *testing.T) {
	s := newTestShell()
	dl := render.NewDisplayContext()
	s.drawPanel(dl, layout.NewBox(layout.Rect(0, 0, 40, 6)), PanelBookmarks)
	out := ansi.Strip(dl.RenderToString(40, 6))
	require.Contains(t, out, "[3]─Remotes─[8]─Tags")
}
