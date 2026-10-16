package jj

// jjj-only command builders (new file in an upstream package; see UPSTREAM.md).

import (
	"strconv"
	"strings"
)

// BookmarkAdvance moves bookmarks forward to `to`. With no names jj advances
// the closest bookmarks; names are matched exactly, not as globs.
func BookmarkAdvance(to string, names ...string) CommandArgs {
	args := CommandArgs{"bookmark", "advance", "--to", to}
	for _, n := range names {
		args = append(args, "exact:"+n)
	}
	return args
}

// FileAnnotate takes a plain path (jj file annotate does not accept filesets).
func FileAnnotate(revision string, file FileName) CommandArgs {
	return CommandArgs{"file", "annotate", "-r", revision, "--color", "always", file.Path()}
}

func ResolveFiles(revision string, files []FileName) CommandArgs {
	return withFilesets(CommandArgs{"resolve", "-r", revision}, files)
}

func DiffEditFiles(revision string, files []FileName) CommandArgs {
	return withFilesets(DiffEdit(revision), files)
}

func withFilesets(args CommandArgs, files []FileName) CommandArgs {
	for _, f := range files {
		args = append(args, f.Escaped())
	}
	return args
}

// BookmarkOnlyIn lists one line per commit reachable from bookmark `name`
// on side `to` but not from side `from`; a side is a remote name, or "" for
// the local bookmark. Counting the lines gives ahead/behind.
func BookmarkOnlyIn(name, from, to string) CommandArgs {
	ref := func(remote string) string {
		if remote == "" {
			return strconv.Quote(name)
		}
		return strconv.Quote(name) + "@" + strconv.Quote(remote)
	}
	return CommandArgs{"log", "--no-graph", "--ignore-working-copy", "--color", "never",
		"-r", ref(from) + ".." + ref(to), "-T", `"x\n"`}
}

// GitRemoteListReadOnly is GitRemoteList without a working-copy snapshot.
func GitRemoteListReadOnly() CommandArgs {
	return CommandArgs{"git", "remote", "list", "--ignore-working-copy", "--color", "never"}
}

// TagListJournal lists local tags for the Tags panel, one per line:
// name;present;commit id;committer time;foreign;subject. foreign is true
// when the tag's commit is reachable from a foreign remote's bookmarks
// (tags/ParseTags drops those); the subject comes last as it may hold ';'.
func TagListJournal(foreign []string) CommandArgs {
	revset := "none()"
	if len(foreign) > 0 {
		parts := make([]string, len(foreign))
		for i, r := range foreign {
			parts[i] = "remote_bookmarks(remote=exact:" + strconv.Quote(r) + ")"
		}
		revset = "::(" + strings.Join(parts, " | ") + ")"
	}
	template := `name ++ ";" ++ present ++ ";" ++ if(normal_target, ` +
		`normal_target.commit_id().shortest() ++ ";" ++ ` +
		`normal_target.committer().timestamp().format("%Y-%m-%dT%H:%M:%S%z") ++ ";" ++ ` +
		`normal_target.contained_in('` + revset + `') ++ ";" ++ ` +
		`normal_target.description().first_line(), ";;false;") ++ "\n"`
	return CommandArgs{"tag", "list", "--ignore-working-copy", "--color", "never", "-T", template}
}
