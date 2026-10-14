package jjj

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/intents"
)

func bindingFor(t *testing.T, c *config.Config, action string) config.BindingConfig {
	t.Helper()
	for _, b := range c.Bindings {
		if b.Action == action {
			return b
		}
	}
	t.Fatalf("no default binding for %s", action)
	return config.BindingConfig{}
}

func TestDefaultsBindShellKeys(t *testing.T) {
	upstream := &config.Config{}
	Current = &Config{}
	require.NoError(t, ApplyDefaults(upstream))

	b := bindingFor(t, upstream, "jjj.shell.focus_files")
	require.Equal(t, "jjj.shell", b.Scope)

	intent, ok := actions.ResolveIntent(actions.ScopeJjjShell, keybindings.Action("jjj.shell.focus_files"), nil)
	require.True(t, ok)
	require.Equal(t, intents.ShellFocusPanel{Panel: 2}, intent)

	intent, ok = actions.ResolveIntent(actions.ScopeJjjBookmarks, keybindings.Action("jjj.bookmarks.advance_to_working_copy"), nil)
	require.True(t, ok)
	require.Equal(t, intents.BookmarkPaneAdvance{ToWorkingCopy: true}, intent)
}

func TestEveryJjjActionHasADefaultKey(t *testing.T) {
	upstream := &config.Config{}
	Current = &Config{}
	require.NoError(t, ApplyDefaults(upstream))

	for _, action := range []string{
		"jjj.shell.focus_detail", "jjj.shell.focus_status", "jjj.shell.focus_files",
		"jjj.shell.focus_bookmarks", "jjj.shell.focus_journal", "jjj.shell.focus_notes",
		"jjj.shell.next_panel", "jjj.shell.prev_panel", "jjj.shell.zoom_in", "jjj.shell.zoom_out",
		"jjj.shell.toggle_journal", "jjj.shell.next_day", "jjj.shell.prev_day",
		"jjj.detail.scroll_down", "jjj.detail.scroll_up", "jjj.detail.half_page_down",
		"jjj.detail.half_page_up", "jjj.detail.page_down", "jjj.detail.page_up", "jjj.detail.back",
		"jjj.notes.scroll_down", "jjj.notes.scroll_up", "jjj.notes.describe", "jjj.notes.back",
		"jjj.status.back",
		"jjj.files.show_diff", "jjj.files.diffedit", "jjj.files.open_editor", "jjj.files.annotate",
		"jjj.files.resolve", "jjj.files.next_panel", "jjj.files.prev_panel",
		"jjj.bookmarks.advance", "jjj.bookmarks.advance_to_working_copy",
	} {
		bindingFor(t, upstream, action)
	}
}
