package jj

// jjj-only command builders (new file in an upstream package; see UPSTREAM.md).

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
