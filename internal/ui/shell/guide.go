package shell

import (
	"strings"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/ui/help"
)

// task is one "common tasks" line in the ? overlay: what an action is for,
// in a sentence. The key comes from the live bindings, so a rebind shows.
type task struct {
	scope, action, desc string
}

type taskGroup struct {
	name  string
	note  string
	tasks []task
}

var guide = []taskGroup{
	{
		name: "Common tasks · Journal (4)",
		note: "jj has no separate amend: edits in the working copy already belong to the current change. To amend an older commit, squash or absorb into it.",
		tasks: []task{
			{"revisions", "revisions.open_squash", "Squash: move the selected change's edits into its parent, or into a target you pick with t. This is how you amend an earlier commit."},
			{"revisions", "revisions.open_absorb", "Absorb: send each changed line to the commit that last touched it. Quick fix-ups across a stack."},
			{"revisions", "revisions.open_inline_describe", "Describe: edit the commit message in place."},
			{"revisions", "revisions.describe", "Describe in $EDITOR."},
			{"revisions", "revisions.new", "New: start an empty change on top of the selected one."},
			{"revisions", "revisions.commit", "Commit: describe the working copy and start a new change after it."},
			{"revisions", "revisions.split", "Split: divide a change in two by choosing files or hunks."},
			{"revisions", "revisions.open_rebase", "Rebase: move a change, its branch or its descendants somewhere else."},
			{"revisions", "revisions.open_abandon", "Abandon: drop a change; its descendants are kept."},
			{"revisions", "revisions.edit", "Edit: make the selected change the working copy."},
			{"revisions", "revisions.diff_edit", "Diff edit: change a commit's contents in your diff editor."},
			{"revisions", "revisions.open_set_bookmark", "Set bookmark: point a bookmark at the selected change."},
			{"revisions", "ui.open_git", "Git: fetch from and push to remotes."},
			{"revisions", "ui.open_undo", "Undo the last jj operation. Repeat to go further back."},
			{"revisions", "ui.exec_jj", "Run any jj command."},
		},
	},
	{
		name: "Common tasks · Bookmarks (3)",
		tasks: []task{
			{"jjj.bookmarks", "jjj.bookmarks.advance", "Advance: move bookmarks forward to the change selected in Journal."},
			{"jjj.bookmarks", "jjj.bookmarks.advance_to_working_copy", "Advance to the working copy, then push (asks first)."},
			{"bookmark_pane", "bookmark_pane.cycle_remotes", "Show one remote's bookmarks at a time (all, local, then each remote)."},
			{"bookmark_pane", "bookmark_pane.show_in_revision", "Show the bookmark's commit in Journal."},
			{"bookmark_pane", "bookmark_pane.fetch", "Fetch from remotes."},
		},
	},
}

// Guide builds the ? overlay's "common tasks" groups from bindings. Tasks
// whose action is unbound are left out.
func Guide(bindings []config.BindingConfig) []help.ScopeGroup {
	var groups []help.ScopeGroup
	for _, g := range guide {
		var entries []help.Entry
		if g.note != "" {
			entries = append(entries, help.Entry{Desc: g.note})
		}
		for _, t := range g.tasks {
			if label := boundKeys(bindings, t.scope, t.action); label != "" {
				entries = append(entries, help.Entry{Label: label, Desc: t.desc})
			}
		}
		groups = append(groups, help.ScopeGroup{Name: g.name, Entries: entries, Wide: true})
	}
	return groups
}

func boundKeys(bindings []config.BindingConfig, scope, action string) string {
	var labels []string
	for _, b := range bindings {
		if b.Scope == scope && b.Action == action && len(b.Args) == 0 {
			if l := help.BindingLabel(b); l != "" {
				labels = append(labels, l)
			}
		}
	}
	return strings.Join(labels, " ")
}
