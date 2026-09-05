// Package journal turns jj log output into a time-sorted, day-grouped list.
package journal

// RowTemplateGraph is the compiled-in row template for graph mode: one line
// per change with the full local date. `format_short_change_id_with_change_offset`
// and `description_placeholder` are jj built-in template aliases (jj >= 0.37).
const RowTemplateGraph = `separate(" ", format_short_change_id_with_change_offset(self), committer.timestamp().local().format("%a %Y-%m-%d %H:%M"), bookmarks, tags, working_copies, if(conflict, label("conflict", "conflict")), if(empty, label("empty", "(empty)")), if(description, description.first_line(), label(if(empty, "empty"), if(current_working_copy, "(working copy)", description_placeholder)))) ++ "\n"`

// RowTemplateJournal is the journal-mode row: time only, because the date is
// carried by the day header above the first row of each day.
const RowTemplateJournal = `separate(" ", format_short_change_id_with_change_offset(self), committer.timestamp().local().format("%H:%M"), bookmarks, tags, working_copies, if(conflict, label("conflict", "conflict")), if(empty, label("empty", "(empty)")), if(description, description.first_line(), label(if(empty, "empty"), if(current_working_copy, "(working copy)", description_placeholder)))) ++ "\n"`
