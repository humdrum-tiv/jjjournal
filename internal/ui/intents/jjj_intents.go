package intents

// jjj (jjjournal) intents. Everything in this file is jjj-only: upstream code
// never references these types. Regenerate the catalog after editing:
// go run ./cmd/genactions. See UPSTREAM.md.

//jjui:bind scope=jjj.shell action=focus_detail set=Panel:0
//jjui:bind scope=jjj.shell action=focus_status set=Panel:1
//jjui:bind scope=jjj.shell action=focus_files set=Panel:2
//jjui:bind scope=jjj.shell action=focus_bookmarks set=Panel:3
//jjui:bind scope=jjj.shell action=focus_journal set=Panel:4
//jjui:bind scope=jjj.shell action=focus_notes set=Panel:6
//jjui:bind scope=jjj.detail action=back set=Panel:4
//jjui:bind scope=jjj.notes action=back set=Panel:4
//jjui:bind scope=jjj.status action=back set=Panel:4
//jjui:bind scope=jjj.tags action=back set=Panel:4
type ShellFocusPanel struct {
	Panel int
}

func (ShellFocusPanel) isIntent() {}

//jjui:bind scope=jjj.shell action=next_panel set=Delta:1
//jjui:bind scope=jjj.shell action=prev_panel set=Delta:-1
//jjui:bind scope=jjj.files action=next_panel set=Delta:1
//jjui:bind scope=jjj.files action=prev_panel set=Delta:-1
type ShellCyclePanel struct {
	Delta int
}

func (ShellCyclePanel) isIntent() {}

//jjui:bind scope=jjj.shell action=zoom_in set=Delta:1
//jjui:bind scope=jjj.shell action=zoom_out set=Delta:-1
type ShellZoom struct {
	Delta int
}

func (ShellZoom) isIntent() {}

//jjui:bind scope=jjj.shell action=toggle_journal
type ShellToggleJournal struct{}

func (ShellToggleJournal) isIntent() {}

//jjui:bind scope=jjj.shell action=next_day set=Forward:true
//jjui:bind scope=jjj.shell action=prev_day set=Forward:false
type ShellJumpDay struct {
	Forward bool
}

func (ShellJumpDay) isIntent() {}

//jjui:bind scope=jjj.detail action=scroll_down set=Delta:1
//jjui:bind scope=jjj.detail action=scroll_up set=Delta:-1
//jjui:bind scope=jjj.detail action=half_page_down set=Delta:1,Half:true
//jjui:bind scope=jjj.detail action=half_page_up set=Delta:-1,Half:true
//jjui:bind scope=jjj.detail action=page_down set=Delta:1,Page:true
//jjui:bind scope=jjj.detail action=page_up set=Delta:-1,Page:true
//jjui:bind scope=jjj.notes action=scroll_down set=Delta:1
//jjui:bind scope=jjj.notes action=scroll_up set=Delta:-1
type PanelScroll struct {
	Delta int
	Half  bool
	Page  bool
}

func (PanelScroll) isIntent() {}

//jjui:bind scope=jjj.notes action=describe
type NotesDescribe struct{}

func (NotesDescribe) isIntent() {}

//jjui:bind scope=jjj.files action=show_diff
type FilesShowDiff struct{}

func (FilesShowDiff) isIntent() {}

//jjui:bind scope=jjj.files action=diffedit
type FilesDiffEdit struct{}

func (FilesDiffEdit) isIntent() {}

//jjui:bind scope=jjj.files action=open_editor
type FilesOpenEditor struct{}

func (FilesOpenEditor) isIntent() {}

//jjui:bind scope=jjj.files action=annotate
type FilesAnnotate struct{}

func (FilesAnnotate) isIntent() {}

//jjui:bind scope=jjj.files action=resolve
type FilesResolve struct{}

func (FilesResolve) isIntent() {}

//jjui:bind scope=jjj.bookmarks action=advance set=ToWorkingCopy:false
//jjui:bind scope=jjj.bookmarks action=advance_to_working_copy set=ToWorkingCopy:true
type BookmarkPaneAdvance struct {
	ToWorkingCopy bool
}

func (BookmarkPaneAdvance) isIntent() {}

//jjui:bind scope=jjj.tags action=move_down set=Delta:1
//jjui:bind scope=jjj.tags action=move_up set=Delta:-1
type TagsMove struct {
	Delta int
}

func (TagsMove) isIntent() {}

//jjui:bind scope=jjj.tags action=reveal
type TagsReveal struct{}

func (TagsReveal) isIntent() {}
