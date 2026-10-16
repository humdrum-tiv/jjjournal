package tags

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForeignRemotes(t *testing.T) {
	out := "main;.;true;false;false;false;a2\n" +
		"main;origin;true;true;false;false;a2\n" +
		"main;git;true;true;false;false;a2\n" +
		"main;upstream;true;false;false;false;b1\n" +
		"feat;zeta;true;false;false;false;c1\n"
	require.Equal(t, []string{"upstream", "zeta"}, ForeignRemotes(out))
	require.Empty(t, ForeignRemotes(""))
}

func TestParseTagsNewestFirstAndOwnOnly(t *testing.T) {
	out := "v0.1;true;aa;2026-09-01T10:00:00+0000;false;first milestone\n" +
		"v0.10.0;true;bb;2026-10-01T10:00:00+0000;true;borrowed release\n" +
		"v0.2;true;cc;2026-10-10T10:00:00+0000;false;second milestone\n" +
		"gone;false;dd;2026-10-11T10:00:00+0000;false;deleted tag"
	got := ParseTags(out)
	require.Len(t, got, 2)
	require.Equal(t, "v0.2", got[0].Name)
	require.Equal(t, "cc", got[0].CommitID)
	require.Equal(t, "second milestone", got[0].Subject)
	require.True(t, got[0].When.Equal(time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC)))
	require.Equal(t, "v0.1", got[1].Name)
}

func TestParseTagsConflicted(t *testing.T) {
	got := ParseTags("knot;true;;;false;\nv0.1;true;aa;2026-09-01T10:00:00+0000;false;first\n")
	require.Len(t, got, 2)
	require.Equal(t, "v0.1", got[0].Name)
	require.Equal(t, Tag{Name: "knot", Conflict: true}, got[1])
}

func TestParseTagsSubjectWithSemicolon(t *testing.T) {
	got := ParseTags("v1;true;aa;2026-09-01T10:00:00+0000;false;fix: a; b; c")
	require.Equal(t, "fix: a; b; c", got[0].Subject)
}
