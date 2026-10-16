package jj

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBookmarkAdvance(t *testing.T) {
	require.Equal(t, CommandArgs{"bookmark", "advance", "--to", "abc"}, BookmarkAdvance("abc"))
	require.Equal(t, CommandArgs{"bookmark", "advance", "--to", "@", "exact:main", "exact:feat/x"},
		BookmarkAdvance("@", "main", "feat/x"))
}

func TestFileCommands(t *testing.T) {
	f := []FileName{NewFileName("a b.go"), NewFileName("c.go")}
	require.Equal(t, CommandArgs{"file", "annotate", "-r", "abc", "--color", "always", "a b.go"}, FileAnnotate("abc", f[0]))
	require.Equal(t, CommandArgs{"resolve", "-r", "abc", `file:"a b.go"`, `file:"c.go"`}, ResolveFiles("abc", f))
	require.Equal(t, CommandArgs{"diffedit", "-r", "abc", `file:"a b.go"`, `file:"c.go"`}, DiffEditFiles("abc", f))
}

func TestBookmarkOnlyIn(t *testing.T) {
	require.Equal(t, CommandArgs{"log", "--no-graph", "--ignore-working-copy", "--color", "never",
		"-r", `"main"@"origin".."main"`, "-T", `"x\n"`}, BookmarkOnlyIn("main", "origin", ""))
	require.Equal(t, `"main".."main"@"origin"`, BookmarkOnlyIn("main", "", "origin")[6])
}
