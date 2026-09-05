package jjj

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultsSetsJournalDefaults(t *testing.T) {
	upstream := &config.Config{}
	Current = &Config{}
	require.NoError(t, ApplyDefaults(upstream))

	require.Equal(t, "all()", upstream.Revisions.Revset)
	require.Equal(t, journal.RowTemplateGraph, upstream.Revisions.Template)
	require.False(t, upstream.Preview.ShowAtStart)
	require.Equal(t, []string{"show", "--color", "always", "-r", "$change_id", "--stat", "--git"}, upstream.Preview.RevisionCommand)
	require.Equal(t, []string{"diff", "--color", "always", "-r", "$change_id", "--git", "$file"}, upstream.Preview.FileCommand)

	require.True(t, Current.Journal.Enabled)
	require.Equal(t, 5000, Current.Journal.Limit)
	require.Equal(t, "~/Humdrum/Daily/{{date}}.md", Current.Journal.DailyNote)
	require.Equal(t, "journal", Current.Journal.Callout)
	require.True(t, Current.Journal.CreateNote)
	require.True(t, Current.Shell.Enabled)
	require.Equal(t, 120, Current.Shell.NarrowColumns)
	require.InDelta(t, 0.34, Current.Shell.LeftWidth, 0.001)
	require.Equal(t, 8, Current.Shell.NotesRows)
	require.Empty(t, Current.Shell.Hidden)
	require.True(t, Current.Tasks.Enabled)
	require.False(t, Current.Tasks.ShowDone)
	require.Equal(t, 26, Current.Stats.Weeks)
}

func TestLoadOverridesOnlyGivenKeys(t *testing.T) {
	Current = &Config{}
	require.NoError(t, ApplyDefaults(&config.Config{}))
	require.NoError(t, Current.Load("[journal]\nlimit = 10\n[shell]\nhidden = [\"files\"]\n"))
	require.Equal(t, 10, Current.Journal.Limit)
	require.Equal(t, []string{"files"}, Current.Shell.Hidden)
	require.True(t, Current.Journal.Enabled, "untouched keys keep their defaults")
	require.Equal(t, 26, Current.Stats.Weeks)
}

func TestLoadIgnoresUpstreamTables(t *testing.T) {
	Current = &Config{}
	require.NoError(t, Current.Load("[revisions]\nrevset = \"mine()\"\n[ui]\ntheme = \"x\"\n"))
}
