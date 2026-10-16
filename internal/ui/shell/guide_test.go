package shell

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/config"
)

func TestGuideTakesKeysFromBindings(t *testing.T) {
	bindings := []config.BindingConfig{
		{Action: "revisions.open_squash", Scope: "revisions", Key: config.StringList{"S"}},
		{Action: "revisions.open_absorb", Scope: "revisions", Key: config.StringList{"x"}},
	}
	groups := Guide(bindings)
	require.Len(t, groups, 2)
	journal := groups[0]
	require.True(t, journal.Wide)
	require.Equal(t, "", journal.Entries[0].Label, "note first")
	require.Equal(t, "S", journal.Entries[1].Label)
	require.Equal(t, "x", journal.Entries[2].Label, "rebinds show")
	require.Len(t, journal.Entries, 3, "unbound tasks are left out")
	require.Empty(t, groups[1].Entries)
}
