# jjj P2a — Scoped keys and the Files / Bookmarks / Notes panels — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every jjj panel its own rebindable keys through upstream's scope/intent dispatch, and add three live main panels — Files (2), Bookmarks (3, with bookmark advance) and Notes (6) — with an accordion layout, zoom, and a narrow-terminal fallback.

**Architecture:** Two optional-interface seams in `internal/ui/ui.go` let the shell extension contribute dispatch scopes (placed before the revisions scopes) and a selection (placed before the revisions selection). The shell keeps a registry of panels; each panel draws itself, receives forwarded non-input messages, and may expose scopes. Files hosts upstream's `operations/details.Operation` persistently; Bookmarks hosts upstream's `bookmarkpane.Model` persistently plus a new `Advance` method in a new file; Notes is a new small model. jjj intents live in a new file inside `internal/ui/intents/` so upstream's action generator picks them up; default keys live in `internal/jjj/defaults.toml`.

**Tech Stack:** Go (module `github.com/idursun/jjui`), Bubble Tea v2 / Lipgloss v2 (`charm.land/*`), `github.com/charmbracelet/x/ansi`, `stretchr/testify`, `jj` 0.45 on PATH.

**Spec:** `docs/superpowers/specs/2026-10-13-jjj-functional-design.md` — this plan implements §3 (foundation), §4 (panels) and §7's `[shell]` keys that already exist. P2b (§5, palette/hint bar) and P2c (§6, review) are later plans. Base spec: `docs/superpowers/specs/2026-09-04-jjjournal-design.md`.

**Deviations from the spec, decided while planning (tell the reviewer):**

1. **No `Extension.Editing()` seam.** Spec §3.1 adds it so raw `HandleKey` is not offered during panel editing. This plan makes `HandleKey` inert (every shell key becomes an action), so the clause has nothing to guard. One seam fewer in `ui.go`.
2. **Seams are optional interfaces, not new `Extension` methods.** `ui.go` type-asserts the extension against the existing `common.ScopeProvider` and `common.SelectionProvider`; the `Extension` interface is unchanged.
3. **Files tree view (`t`) and the Bookmarks `@` marker are deferred** to backlog follow-ups (Task 8 files them). Both need edits to existing upstream rendering code (`details_list.go`, `bookmarkpane/view.go`); the bookmark pane already sorts bookmarks at the selected revision to the top.
4. **Files resolve is `M` (shift+m), not `m`:** upstream's file-details scope already binds `m` to toggle-select. Resolve runs on any file; jj itself reports "no conflicts" for a clean one.
5. **Files hosts upstream's modal `details.Operation` persistently rather than `DetailsList` (spec §4.2);** the shell compensates for its modal assumptions (final-review fixes F1/F4: a `common.Close` from the hosted op also clears its confirmation, focus cannot leave an editing panel, confirmation messages reach only the focused panel). A jjj-side async host over `DetailsList` is a follow-up task.

## Global Constraints

- Every VCS mutation goes through `jj` (a hook blocks mutating `git`). Commit with `jj commit -m "..."`. Never `git add` / `git commit`.
- Upstream files may be edited only for the seams named here; every touched upstream file is listed in `UPSTREAM.md` with the reason. No reformatting, renaming or moving of upstream code. New files inside upstream packages are allowed and also listed.
- `make check` (vet, build, `go build ./cmd/jjui`, `go test ./...`) must pass after every task.
- Never hand-edit generated files (`internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua`); regenerate with `go run ./cmd/genactions`.
- Models handle `intents.Intent`, not raw `tea.KeyMsg` (AGENTS.md). In `HandleIntent`, return `handled = true` whenever the scope consumes the intent.
- Hosted upstream models (`details.Operation`, `bookmarkpane.Model`) never receive `tea.KeyMsg`/`tea.MouseMsg` or `intents.Intent` values through the shell's message forwarding; keys reach them only through dispatch.
- The repo root is `/Users/kortum/Developer/Home/jjjournal`; run commands from there. No new module dependencies.
- Scope names: `jjj.shell`, `jjj.files`, `jjj.bookmarks`, `jjj.detail`, `jjj.notes`, `jjj.status` (generated constants `actions.ScopeJjjShell`, `actions.ScopeJjjFiles`, `actions.ScopeJjjBookmarks`, `actions.ScopeJjjDetail`, `actions.ScopeJjjNotes`, `actions.ScopeJjjStatus`).

## Review Focus

1. **Holding `j` in Journal with Files on screen** must not run `jj status` once per row: Files debounces selection changes and ignores stale results. Pinned by `TestFilesIgnoresStaleRevision` (Task 6).
2. **An `intents.Intent` travelling as a plain message** (e.g. `intents.Cancel{}` from some `intents.Invoke`) must not be handled by the hosted Files op or bookmark pane, where it would close a confirmation or clear a filter. Pinned by `TestUpdateDoesNotForwardIntents` (Task 4).
3. **Typing in a panel's filter or answering its confirmation**: `2`, `tab`, `J` go to the input, not to the shell. Pinned by `TestScopesDropShellWhileEditing` (Task 6) and `TestBookmarksScopesWhileEditing` (Task 7).
4. **`[shell] hidden = ["files"]` and very short terminals:** the hidden panel is not focusable by digit or tab, and accordion heights always sum to the column height. Pinned by `TestHiddenPanelIsNotFocusable` (Task 4) and `TestAccordionAlwaysSumsToHeight` (Task 3).
5. **An upstream operation open on the Journal (rebase, inline describe, quick search):** the shell contributes no scopes, so digits and `tab` reach the operation. Pinned by `TestScopesEmptyOutsideNormalMode` (Task 4).

---

## File structure

New files:

| Path | Responsibility |
|---|---|
| `internal/ui/ui_extension_test.go` | tests for the two `ui.go` seams |
| `internal/ui/intents/jjj_intents.go` | every jjj intent type + `//jjui:bind` annotations |
| `internal/jjj/bindings_test.go` | jjj default bindings load and resolve to jjj intents |
| `internal/ui/shell/panels.go` | `panel` interface, the shell's scope handler, Journal / Status / Detail panels |
| `internal/ui/notes/notes.go`, `notes_test.go` | Notes model: description fetch, wrap, scroll |
| `internal/ui/files/files.go`, `files_test.go` | Files model: hosts `details.Operation`, debounce, jjj file actions |
| `internal/ui/shell/bookmarks.go` | Bookmarks panel wrapper around `bookmarkpane.Model` |
| `internal/ui/bookmarkpane/advance.go`, `advance_test.go` | `Model.Advance` (new file in an upstream package) |
| `internal/jj/jjj_commands.go`, `jjj_commands_test.go` | `jj.BookmarkAdvance`, `jj.FileAnnotate`, `jj.ResolveFiles`, `jj.DiffEditFiles` (new file in an upstream package) |

Modified:

| Path | Change |
|---|---|
| `internal/ui/ui.go` | seams: extension scopes in `dispatchScopes`, extension selection in `selectionProviders` |
| `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua` | regenerated |
| `internal/jjj/defaults.toml` | `bindings = [...]` for jjj scopes |
| `internal/ui/shell/layout.go`, `layout_test.go` | accordion, zoom, presence-aware layout and tab cycle |
| `internal/ui/shell/shell.go`, `shell_test.go` | panel registry, scopes, intent handling, render; raw keys removed |
| `UPSTREAM.md`, `docs/jjj.md` | upstream record, key reference |

---

### Task 1: `ui.go` seams — extension scopes and selection

**Files:**
- Modify: `internal/ui/ui.go` (`selectionProviders` ~line 167, `dispatchScopes` ~line 586)
- Create: `internal/ui/ui_extension_test.go`
- Modify: `UPSTREAM.md`

**Interfaces:**
- Produces: when the extension passed to `ui.NewWithExtension` also implements `common.ScopeProvider`, its `Scopes()` are inserted into `dispatchScopes()` immediately before the revisions scopes — and only on the branch where revisions is the primary model (no stacked view, no oplog). When it implements `common.SelectionProvider`, its `Selection()` is consulted after stacked/oplog and before revisions; the first non-nil `Highlighted` wins.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/ui_extension_test.go`:

```go
package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/input"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/test"
)

// fakeExt is a minimal Extension that also provides scopes and a selection.
type fakeExt struct {
	scopes    []common.Scope
	selection common.SelectionSnapshot
}

func (f *fakeExt) Attach(Core)                                              {}
func (f *fakeExt) Init() tea.Cmd                                            { return nil }
func (f *fakeExt) Update(tea.Msg) (tea.Cmd, bool)                           { return nil, false }
func (f *fakeExt) HandleKey(tea.KeyMsg) (tea.Cmd, bool)                     { return nil, false }
func (f *fakeExt) Render(*render.DisplayContext, layout.Box, Core) bool     { return false }
func (f *fakeExt) Scopes() []common.Scope                                   { return f.scopes }
func (f *fakeExt) Selection() common.SelectionSnapshot                      { return f.selection }

type fakeHandler struct{}

func (fakeHandler) HandleIntent(intents.Intent) (tea.Cmd, bool) { return nil, false }
func (fakeHandler) Update(tea.Msg) tea.Cmd                      { return nil }

func newExtModel(t *testing.T, ext *fakeExt) *Model {
	t.Helper()
	ctx := test.NewTestContext(test.NewTestCommandRunner(t))
	m := NewUI(ctx)
	m.ext = ext
	return m
}

func TestExtensionScopesPrecedeRevisions(t *testing.T) {
	ext := &fakeExt{scopes: []common.Scope{{Name: "jjj.fake", Leak: common.LeakAll, Handler: fakeHandler{}}}}
	m := newExtModel(t, ext)

	names := scopeNames(m.dispatchScopes())

	fake := indexOf(names, "jjj.fake")
	revs := indexOf(names, "revisions")
	require.GreaterOrEqual(t, fake, 0, "extension scope present: %v", names)
	require.Greater(t, revs, fake, "extension scope before revisions: %v", names)
}

func TestExtensionScopesAbsentWhileStackedViewOpen(t *testing.T) {
	ext := &fakeExt{scopes: []common.Scope{{Name: "jjj.fake", Leak: common.LeakAll, Handler: fakeHandler{}}}}
	m := newExtModel(t, ext)
	m.stacked = input.NewWithTitle("Prompt", "Text: ", "")

	assert.Equal(t, -1, indexOf(scopeNames(m.dispatchScopes()), "jjj.fake"))
}

func TestExtensionSelectionWinsOverRevisions(t *testing.T) {
	file := common.SelectedFile{ChangeId: "abc", CommitId: "123", File: jj.NewFileName("a.go")}
	m := newExtModel(t, &fakeExt{selection: common.SelectionSnapshot{Highlighted: file}})

	assert.Equal(t, file, m.selectionSnapshot().Highlighted)
}

func TestExtensionEmptySelectionFallsThrough(t *testing.T) {
	m := newExtModel(t, &fakeExt{})
	// No revisions loaded either: the snapshot is simply empty, not a panic.
	assert.Nil(t, m.selectionSnapshot().Highlighted)
}

func scopeNames(scopes []common.Scope) []keybindings.ScopeName {
	out := make([]keybindings.ScopeName, len(scopes))
	for i, s := range scopes {
		out[i] = s.Name
	}
	return out
}

func indexOf(names []keybindings.ScopeName, name keybindings.ScopeName) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ui -run 'TestExtension' -v`
Expected: FAIL — `jjj.fake` missing from the scopes and the selection is nil.

- [ ] **Step 3: Implement the seams**

In `internal/ui/ui.go`, in `selectionProviders`, insert before `if m.revisions != nil {`:

```go
	// jjj: an Extension may contribute the highlighted item (e.g. a file in
	// a shell panel); it ranks below modal views and above the revisions list.
	if provider, ok := m.ext.(common.SelectionProvider); ok {
		providers = append(providers, provider)
	}
```

In `dispatchScopes`, change the final `else` branch from:

```go
	} else {
		scopes = append(scopes, m.splitScopes(m.revisions.Scopes())...)
	}
```

to:

```go
	} else {
		// jjj: Extension scopes sit just inside the revisions scopes, so a
		// focused shell panel can claim keys (and block revisions with
		// LeakGlobal) while modals above still win.
		if provider, ok := m.ext.(common.ScopeProvider); ok {
			scopes = append(scopes, provider.Scopes()...)
		}
		scopes = append(scopes, m.splitScopes(m.revisions.Scopes())...)
	}
```

(`m.ext` may be nil; a type assertion on a nil interface yields `ok == false`.)

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/ui -run 'TestExtension' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Record the seam in `UPSTREAM.md`**

Replace the `internal/ui/ui.go` row of the "Touched upstream files" table with:

```markdown
| `internal/ui/ui.go` | `Extension` hook (`Extension`, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`) and its three call sites in `Init`/`Update`/`View`; `Update`'s return value became a named result (`cmd tea.Cmd`) with a deferred `cmd = tea.Batch(extCmd, cmd)` so the extension's command is folded into every return path, not just one. P2a: an extension implementing `common.ScopeProvider` has its scopes inserted before the revisions scopes in `dispatchScopes` (revisions-primary branch only), and one implementing `common.SelectionProvider` is consulted in `selectionProviders` before revisions |
```

- [ ] **Step 6: Verify and commit**

Run: `make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): ui seams for extension scopes and selection (TASK-012.01)"
```

---

### Task 2: jjj intents, generated catalog, default bindings

**Files:**
- Create: `internal/ui/intents/jjj_intents.go`
- Regenerate: `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua`
- Modify: `internal/jjj/defaults.toml`
- Create: `internal/jjj/bindings_test.go`
- Modify: `UPSTREAM.md`

**Interfaces:**
- Produces (package `intents`): `ShellFocusPanel{Panel int}`, `ShellCyclePanel{Delta int}`, `ShellZoom{Delta int}`, `ShellToggleJournal{}`, `ShellJumpDay{Forward bool}`, `PanelScroll{Delta int; Half bool; Page bool}`, `NotesDescribe{}`, `FilesShowDiff{}`, `FilesDiffEdit{}`, `FilesOpenEditor{}`, `FilesAnnotate{}`, `FilesResolve{}`, `BookmarkPaneAdvance{ToWorkingCopy bool}`.
- Produces (package `actions`, generated): `ScopeJjjShell`, `ScopeJjjFiles`, `ScopeJjjBookmarks`, `ScopeJjjDetail`, `ScopeJjjNotes`, `ScopeJjjStatus`; action IDs `jjj.<scope>.<action>`.

- [ ] **Step 1: Write the failing test**

Create `internal/jjj/bindings_test.go`:

```go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/jjj -run 'TestDefaultsBindShellKeys|TestEveryJjjActionHasADefaultKey' -v`
Expected: FAIL to compile — `actions.ScopeJjjShell` and `intents.ShellFocusPanel` undefined.

- [ ] **Step 3: Add the intents**

Create `internal/ui/intents/jjj_intents.go`:

```go
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
```

- [ ] **Step 4: Regenerate**

Run: `go run ./cmd/genactions`
Expected: exits 0; `git status --short` (read-only) shows the three generated files modified. If it reports a validation error, fix the annotation it names — do not edit generated files.

- [ ] **Step 5: Add the default bindings**

In `internal/jjj/defaults.toml`, insert after the header comment block and **before** `[revisions]` (a top-level key must precede the first table):

```toml
# jjj keys. Scope order at runtime: the focused panel's scope, then
# jjj.shell, then upstream's scopes. Rebind any of these in config.toml.
bindings = [
    # shell (global while no upstream operation is open)
    { key = "0", action = "jjj.shell.focus_detail", scope = "jjj.shell", desc = "focus detail" },
    { key = "1", action = "jjj.shell.focus_status", scope = "jjj.shell", desc = "focus status" },
    { key = "2", action = "jjj.shell.focus_files", scope = "jjj.shell", desc = "focus files" },
    { key = "3", action = "jjj.shell.focus_bookmarks", scope = "jjj.shell", desc = "focus bookmarks" },
    { key = "4", action = "jjj.shell.focus_journal", scope = "jjj.shell", desc = "focus journal" },
    { key = "6", action = "jjj.shell.focus_notes", scope = "jjj.shell", desc = "focus notes" },
    { key = "tab", action = "jjj.shell.next_panel", scope = "jjj.shell", desc = "next panel" },
    { key = "shift+tab", action = "jjj.shell.prev_panel", scope = "jjj.shell", desc = "previous panel" },
    { key = "+", action = "jjj.shell.zoom_in", scope = "jjj.shell", desc = "zoom in" },
    { key = "_", action = "jjj.shell.zoom_out", scope = "jjj.shell", desc = "zoom out" },
    { key = ["J", "shift+j"], action = "jjj.shell.toggle_journal", scope = "jjj.shell", desc = "journal/graph" },
    { key = "}", action = "jjj.shell.next_day", scope = "jjj.shell", desc = "next day" },
    { key = "{", action = "jjj.shell.prev_day", scope = "jjj.shell", desc = "previous day" },

    # detail
    { key = ["j", "down"], action = "jjj.detail.scroll_down", scope = "jjj.detail", desc = "scroll down" },
    { key = ["k", "up"], action = "jjj.detail.scroll_up", scope = "jjj.detail", desc = "scroll up" },
    { key = "ctrl+d", action = "jjj.detail.half_page_down", scope = "jjj.detail", desc = "half page down" },
    { key = "ctrl+u", action = "jjj.detail.half_page_up", scope = "jjj.detail", desc = "half page up" },
    { key = "pgdown", action = "jjj.detail.page_down", scope = "jjj.detail", desc = "page down" },
    { key = "pgup", action = "jjj.detail.page_up", scope = "jjj.detail", desc = "page up" },
    { key = "esc", action = "jjj.detail.back", scope = "jjj.detail", desc = "back to journal" },

    # notes
    { key = ["j", "down"], action = "jjj.notes.scroll_down", scope = "jjj.notes", desc = "scroll down" },
    { key = ["k", "up"], action = "jjj.notes.scroll_up", scope = "jjj.notes", desc = "scroll up" },
    { key = "enter", action = "jjj.notes.describe", scope = "jjj.notes", desc = "describe" },
    { key = "esc", action = "jjj.notes.back", scope = "jjj.notes", desc = "back to journal" },

    # status
    { key = "esc", action = "jjj.status.back", scope = "jjj.status", desc = "back to journal" },

    # files (upstream revisions.details keys also apply: space r s S A d / *)
    { key = "enter", action = "jjj.files.show_diff", scope = "jjj.files", desc = "diff in detail" },
    { key = "e", action = "jjj.files.diffedit", scope = "jjj.files", desc = "diffedit" },
    { key = "o", action = "jjj.files.open_editor", scope = "jjj.files", desc = "open in $EDITOR" },
    { key = "b", action = "jjj.files.annotate", scope = "jjj.files", desc = "annotate" },
    { key = ["M", "shift+m"], action = "jjj.files.resolve", scope = "jjj.files", desc = "resolve" },
    { key = "tab", action = "jjj.files.next_panel", scope = "jjj.files", desc = "next panel" },
    { key = "shift+tab", action = "jjj.files.prev_panel", scope = "jjj.files", desc = "previous panel" },

    # bookmarks (upstream bookmark_pane keys also apply)
    { key = "a", action = "jjj.bookmarks.advance", scope = "jjj.bookmarks", desc = "advance" },
    { key = ["A", "shift+a"], action = "jjj.bookmarks.advance_to_working_copy", scope = "jjj.bookmarks", desc = "advance to @ + push" },
]
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/jjj ./cmd/genactions ./internal/ui/actions -v -run 'TestDefaultsBindShellKeys|TestEveryJjjActionHasADefaultKey|TestGeneratedCatalogIsUpToDate'`
Expected: PASS.

- [ ] **Step 7: Record in `UPSTREAM.md`**

Add rows to the touched-files table:

```markdown
| `internal/ui/intents/jjj_intents.go` | new file: jjj intent types and `//jjui:bind` annotations (the generator scans only this directory); no upstream intent file edited |
| `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua` | regenerated with jjj entries — after every upstream sync run `go run ./cmd/genactions`, never hand-merge |
```

In the sync procedure block, insert `go run ./cmd/genactions` on its own line after the `jj rebase` line and before `make check`.

- [ ] **Step 8: Verify and commit**

Run: `make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): jjj intents and default bindings (TASK-012.01, TASK-006)"
```

---

### Task 3: Presence-aware layout — accordion, zoom, tab cycle

**Files:**
- Modify: `internal/ui/shell/layout.go` (rewrite)
- Modify: `internal/ui/shell/layout_test.go` (rewrite)
- Modify: `internal/ui/shell/shell.go` (`Render` and the two `tab` cases only, to compile against the new layout)

**Interfaces:**
- Produces:
  - `type Zoom int` with `ZoomNone`, `ZoomColumn`, `ZoomFull`, and `func (z Zoom) Step(d int) Zoom` (clamped).
  - `const MinPanelRows = 3`, `const StatusHeight = 5` (unchanged).
  - `var Order = []PanelID{PanelStatus, PanelFiles, PanelBookmarks, PanelJournal, PanelDetail, PanelNotes}` — draw and tab order.
  - `type Params struct { LeftWidth float64; NotesRows int; Focus PanelID; Zoom Zoom; Present map[PanelID]bool }`.
  - `type Layout map[PanelID]layout.Box` and `func Compute(box layout.Box, p Params) Layout`.
  - `func Accordion(h int, panels []PanelID, focus PanelID) []int`.
  - `func Cycle(present map[PanelID]bool, from PanelID, d int) PanelID`.
- Removes: `Layout` struct fields, `cycle`, `PanelID.next/prev`, `stepPanel`.

- [ ] **Step 1: Write the failing tests**

Replace `internal/ui/shell/layout_test.go` with:

```go
package shell

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/require"
)

func all() map[PanelID]bool {
	return map[PanelID]bool{PanelStatus: true, PanelFiles: true, PanelBookmarks: true,
		PanelJournal: true, PanelDetail: true, PanelNotes: true}
}

func TestAccordionFocusedHalfJournalTwoFifths(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	require.Equal(t, []int{10, 10, 20}, Accordion(40, left, PanelJournal))
	require.Equal(t, []int{20, 4, 16}, Accordion(40, left, PanelFiles))
	require.Equal(t, []int{10, 10, 20}, Accordion(40, left, PanelDetail), "focus elsewhere: Journal half")
	require.Equal(t, []int{40}, Accordion(40, []PanelID{PanelJournal}, PanelFiles))
}

func TestAccordionKeepsMinimumRows(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	require.Equal(t, []int{5, 3, 4}, Accordion(12, left, PanelFiles))
	require.Equal(t, []int{2, 2, 3}, Accordion(7, left, PanelFiles), "below 3 rows each: equal split")
}

func TestAccordionAlwaysSumsToHeight(t *testing.T) {
	left := []PanelID{PanelFiles, PanelBookmarks, PanelJournal}
	for h := 0; h <= 80; h++ {
		for _, f := range []PanelID{PanelFiles, PanelBookmarks, PanelJournal, PanelDetail} {
			sum := 0
			for _, v := range Accordion(h, left, f) {
				require.GreaterOrEqual(t, v, 0)
				sum += v
			}
			require.Equal(t, h, sum, "h=%d focus=%d", h, f)
		}
	}
}

func TestComputeAllPanels(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	lay := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelJournal, Present: all()})

	require.Equal(t, 68, lay[PanelStatus].R.Dx())
	require.Equal(t, StatusHeight, lay[PanelStatus].R.Dy())
	require.Equal(t, 45, lay[PanelFiles].R.Dy()+lay[PanelBookmarks].R.Dy()+lay[PanelJournal].R.Dy())
	require.Equal(t, 22, lay[PanelJournal].R.Dy(), "focused Journal: half of 45")
	require.Equal(t, 132, lay[PanelDetail].R.Dx())
	require.Equal(t, 10, lay[PanelNotes].R.Dy(), "8 rows + border")
	require.Equal(t, 40, lay[PanelDetail].R.Dy())
	require.Equal(t, 40, lay[PanelNotes].R.Min.Y)
}

func TestComputeAbsentPanelsTakeNoSpace(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	present := map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
	lay := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelJournal, Present: present})

	_, hasFiles := lay[PanelFiles]
	_, hasNotes := lay[PanelNotes]
	require.False(t, hasFiles)
	require.False(t, hasNotes)
	require.Equal(t, 45, lay[PanelJournal].R.Dy())
	require.Equal(t, 50, lay[PanelDetail].R.Dy())
}

func TestComputeZoom(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	col := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelFiles, Zoom: ZoomColumn, Present: all()})
	require.Equal(t, layout.Rect(0, 0, 68, 50), col[PanelFiles].R)
	_, hasJournal := col[PanelJournal]
	require.False(t, hasJournal, "column zoom hides the other left panels")
	require.Equal(t, 40, col[PanelDetail].R.Dy(), "right column unaffected")

	full := Compute(box, Params{LeftWidth: 0.34, NotesRows: 8, Focus: PanelNotes, Zoom: ZoomFull, Present: all()})
	require.Len(t, full, 1)
	require.Equal(t, box.R, full[PanelNotes].R)
}

func TestComputeClampsLeftWidth(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	present := map[PanelID]bool{PanelJournal: true, PanelDetail: true}
	require.Equal(t, 40, Compute(box, Params{LeftWidth: 0.05, Present: present})[PanelJournal].R.Dx())
	require.Equal(t, 120, Compute(box, Params{LeftWidth: 0.9, Present: present})[PanelJournal].R.Dx())
}

func TestCycleSkipsAbsentPanels(t *testing.T) {
	present := map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
	require.Equal(t, PanelJournal, Cycle(present, PanelStatus, 1))
	require.Equal(t, PanelDetail, Cycle(present, PanelJournal, 1))
	require.Equal(t, PanelStatus, Cycle(present, PanelDetail, 1))
	require.Equal(t, PanelDetail, Cycle(present, PanelStatus, -1))
	require.Equal(t, PanelJournal, Cycle(present, PanelFiles, 1), "unknown start falls back to Journal")
	require.Equal(t, PanelFiles, Cycle(all(), PanelStatus, 1))
}

func TestZoomStepClamps(t *testing.T) {
	require.Equal(t, ZoomColumn, ZoomNone.Step(1))
	require.Equal(t, ZoomFull, ZoomFull.Step(1))
	require.Equal(t, ZoomNone, ZoomNone.Step(-1))
	require.Equal(t, ZoomColumn, ZoomFull.Step(-1))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ui/shell -run 'TestAccordion|TestCompute|TestCycle|TestZoom' -v`
Expected: FAIL to compile — `Accordion`, `Params`, `Cycle`, `ZoomNone` undefined.

- [ ] **Step 3: Rewrite `layout.go`**

Replace `internal/ui/shell/layout.go` with:

```go
// Package shell is jjj's lazygit-style layout: several panels visible at once
// around upstream's revisions list. It plugs into ui.Model through the
// ui.Extension hook; its keys reach it as intents through the scopes it
// contributes (see Shell.Scopes).
package shell

import (
	"slices"

	"github.com/idursun/jjui/internal/ui/layout"
)

// PanelID numbers panels the lazygit way (spec §3.1); the digit is the focus key.
type PanelID int

const (
	PanelDetail    PanelID = 0
	PanelStatus    PanelID = 1
	PanelFiles     PanelID = 2
	PanelBookmarks PanelID = 3
	PanelJournal   PanelID = 4
	PanelTasks     PanelID = 5 // P3
	PanelNotes     PanelID = 6
	PanelStats     PanelID = 7 // P4
)

// Order is the draw order and the tab order.
var Order = []PanelID{PanelStatus, PanelFiles, PanelBookmarks, PanelJournal, PanelDetail, PanelNotes}

// leftFlex are the left column's accordion panels, top to bottom.
var leftFlex = []PanelID{PanelFiles, PanelBookmarks, PanelJournal}

func isLeft(id PanelID) bool { return id == PanelStatus || slices.Contains(leftFlex, id) }

// Zoom is the focused panel's zoom level (spec §4.1): + steps up, _ down.
type Zoom int

const (
	ZoomNone Zoom = iota
	ZoomColumn
	ZoomFull
)

func (z Zoom) Step(d int) Zoom { return Zoom(max(int(ZoomNone), min(int(ZoomFull), int(z)+d))) }

const (
	// StatusHeight is the Status panel's outer height: three lines plus border.
	StatusHeight = 5
	// MinPanelRows is the smallest outer height an accordion panel gets
	// while the column has room for every panel at that size.
	MinPanelRows = 3
)

type Params struct {
	LeftWidth float64
	NotesRows int // Notes' inner rows; its outer height adds the border
	Focus     PanelID
	Zoom      Zoom
	Present   map[PanelID]bool // registered and not hidden
}

// Layout maps each panel drawn this frame to its outer box (border included).
type Layout map[PanelID]layout.Box

// Compute splits box into a left column (Status, then the accordion panels)
// and a right column (Detail above Notes). Zoom gives the focused panel its
// whole column or the whole box. Absent panels take no space.
func Compute(box layout.Box, p Params) Layout {
	lay := Layout{}
	if p.Zoom == ZoomFull {
		lay[p.Focus] = box
		return lay
	}
	lw := min(max(p.LeftWidth, 0.2), 0.6)
	cols := box.H(layout.Percent(int(lw*100+0.5)), layout.Fill(1))
	if len(cols) < 2 {
		lay[PanelJournal] = box
		return lay
	}
	left, right := cols[0], cols[1]
	if p.Zoom == ZoomColumn && isLeft(p.Focus) {
		lay[p.Focus] = left
	} else {
		computeLeft(lay, left, p)
	}
	if p.Zoom == ZoomColumn && !isLeft(p.Focus) {
		lay[p.Focus] = right
	} else {
		computeRight(lay, right, p)
	}
	return lay
}

func computeLeft(lay Layout, col layout.Box, p Params) {
	rest := col
	if p.Present[PanelStatus] {
		var top layout.Box
		top, rest = col.CutTop(StatusHeight)
		lay[PanelStatus] = top
	}
	var panels []PanelID
	for _, id := range leftFlex {
		if id == PanelJournal || p.Present[id] {
			panels = append(panels, id)
		}
	}
	for i, h := range Accordion(rest.R.Dy(), panels, p.Focus) {
		var b layout.Box
		b, rest = rest.CutTop(h)
		lay[panels[i]] = b
	}
}

func computeRight(lay Layout, col layout.Box, p Params) {
	notes := p.NotesRows + 2
	if p.Present[PanelNotes] && p.NotesRows > 0 && col.R.Dy() >= notes+MinPanelRows {
		detail, n := col.CutBottom(notes)
		lay[PanelDetail], lay[PanelNotes] = detail, n
		return
	}
	lay[PanelDetail] = col
}

// Accordion splits h rows among panels (top to bottom). The focused panel
// gets half; an unfocused Journal gets two fifths, or half when focus is
// outside these panels; the others share the rest with at least
// MinPanelRows each, taking rows back from the largest panel if needed.
// Below MinPanelRows per panel it splits equally. The result sums to h.
func Accordion(h int, panels []PanelID, focus PanelID) []int {
	n := len(panels)
	out := make([]int, n)
	switch {
	case n == 0:
		return out
	case n == 1:
		out[0] = h
		return out
	case h < n*MinPanelRows:
		for i := range out {
			out[i] = h / n
		}
		out[n-1] += h - (h/n)*n
		return out
	}

	assigned := make([]bool, n)
	fi, ji := slices.Index(panels, focus), slices.Index(panels, PanelJournal)
	if fi >= 0 {
		out[fi], assigned[fi] = h/2, true
	}
	if ji >= 0 && ji != fi {
		if fi >= 0 {
			out[ji] = h * 2 / 5
		} else {
			out[ji] = h / 2
		}
		assigned[ji] = true
	}

	used, others := 0, 0
	for i := range out {
		used += out[i]
		if !assigned[i] {
			others++
		}
	}
	rem := h - used
	if others == 0 {
		target := ji
		if target < 0 {
			target = fi
		}
		out[target] += rem
		return out
	}
	share := rem / others
	if share < MinPanelRows {
		for need := MinPanelRows*others - rem; need > 0; need-- {
			out[largestAssigned(out, assigned)]--
		}
		rem, share = MinPanelRows*others, MinPanelRows
	}
	last := -1
	for i := range out {
		if !assigned[i] {
			out[i], last = share, i
		}
	}
	out[last] += rem - share*others
	return out
}

func largestAssigned(out []int, assigned []bool) int {
	best := -1
	for i := range out {
		if assigned[i] && (best < 0 || out[i] > out[best]) {
			best = i
		}
	}
	return best
}

// Cycle steps from `from` by d through the present panels in Order.
// An unknown or absent start returns PanelJournal.
func Cycle(present map[PanelID]bool, from PanelID, d int) PanelID {
	var ids []PanelID
	for _, id := range Order {
		if present[id] {
			ids = append(ids, id)
		}
	}
	i := slices.Index(ids, from)
	if i < 0 || len(ids) == 0 {
		return PanelJournal
	}
	return ids[((i+d)%len(ids)+len(ids))%len(ids)]
}
```

- [ ] **Step 4: Adapt `shell.go` to compile**

In `internal/ui/shell/shell.go`:

Replace the two `tab` cases in `HandleKey`:

```go
	case "tab":
		s.setFocus(Cycle(s.present(), s.focus, 1))
		return nil, true
	case "shift+tab":
		s.setFocus(Cycle(s.present(), s.focus, -1))
		return nil, true
```

Add below `setFocus`:

```go
// present is the set of panels this build draws (Task 4 replaces this with
// the panel registry).
func (s *Shell) present() map[PanelID]bool {
	return map[PanelID]bool{PanelStatus: true, PanelJournal: true, PanelDetail: true}
}
```

In `Render`, replace `lay := Compute(content, s.cfg.Shell.LeftWidth)` and the three `s.panel(...)` calls' boxes:

```go
	lay := Compute(content, Params{
		LeftWidth: s.cfg.Shell.LeftWidth,
		NotesRows: s.cfg.Shell.NotesRows,
		Focus:     s.focus,
		Present:   s.present(),
	})

	s.panel(dl, lay[PanelStatus], PanelStatus, "Status", func(inner layout.Box) {
		s.status.ViewRect(dl, inner)
	})
	s.panel(dl, lay[PanelJournal], PanelJournal, "Journal", func(inner layout.Box) {
		rows := inner.V(layout.Fixed(1), layout.Fill(1))
		if len(rows) < 2 {
			return
		}
		core.Revset.ViewRect(dl, rows[0])
		core.RenderMain(rows[1])
	})
	s.panel(dl, lay[PanelDetail], PanelDetail, "Detail", func(inner layout.Box) {
		s.detail.ViewRect(dl, inner)
	})
```

(`panel` already returns early for boxes smaller than 4×3, so a missing map entry — a zero box — draws nothing.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/ui/shell -v`
Expected: PASS — the new layout tests and the existing `TestHandleKey*` tests.

- [ ] **Step 6: Verify and commit**

Run: `make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): accordion layout, zoom levels, presence-aware tab cycle (TASK-012.01)"
```

---

### Task 4: Shell panel registry, scopes and intent handling (raw keys removed)

**Files:**
- Create: `internal/ui/shell/panels.go`
- Modify: `internal/ui/shell/shell.go` (rewrite of `Shell`, `New`, `Init`, `Update`, `HandleKey`, `setFocus`, `present`, `Render`; `Attach` and `rowSource` unchanged)
- Modify: `internal/ui/shell/shell_test.go` (rewrite)

**Interfaces:**
- Consumes: Task 1 seams (the shell implements `common.ScopeProvider` and `common.SelectionProvider`), Task 2 intents and scope constants, Task 3 `Compute`/`Cycle`/`Zoom`.
- Produces (package `shell`):
  - `type panel interface { title() string; init() tea.Cmd; update(msg tea.Msg) tea.Cmd; draw(dl *render.DisplayContext, inner layout.Box); scopes() []common.Scope; setFocused(bool); editing() bool }`
  - optional `type intentPanel interface { handleIntent(intents.Intent) (tea.Cmd, bool) }` and `type selectionPanel interface { selection() common.SelectionSnapshot }`.
  - `type handler struct{ s *Shell }` implementing `common.ScopeHandler`; every jjj scope uses it.
  - `func (s *Shell) register(id PanelID, p panel)`; `s.panels map[PanelID]panel`.
  - `func (s *Shell) Scopes() []common.Scope`, `func (s *Shell) Selection() common.SelectionSnapshot`.
  - `s.normalMode func() bool` — defaults to the core revisions check; tests override it.
  - `s.detailFile *common.SelectedFile` — while set and Detail is focused, Selection reports this file (Task 6 sets it).
  - `func (s *Shell) setFocus(id PanelID)`.

- [ ] **Step 1: Write the failing tests**

Replace `internal/ui/shell/shell_test.go` with:

```go
package shell

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui/actions"
	keybindings "github.com/idursun/jjui/internal/ui/bindings"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

func newTestShell(hidden ...string) *Shell {
	s := New(&appContext.MainContext{}, &jjj.Config{
		Shell: jjj.ShellConfig{Enabled: true, LeftWidth: 0.34, NarrowColumns: 120, NotesRows: 8, Hidden: hidden},
	})
	s.normalMode = func() bool { return true }
	return s
}

func names(scopes []common.Scope) []keybindings.ScopeName {
	out := make([]keybindings.ScopeName, len(scopes))
	for i, sc := range scopes {
		out[i] = sc.Name
	}
	return out
}

func TestFocusIntents(t *testing.T) {
	s := newTestShell()

	_, handled := s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelDetail)})
	require.True(t, handled)
	require.Equal(t, PanelDetail, s.focus)

	s.handleIntent(intents.ShellCyclePanel{Delta: 1})
	require.Equal(t, Cycle(s.present(), PanelDetail, 1), s.focus)

	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelJournal)})
	s.handleIntent(intents.ShellZoom{Delta: 1})
	require.Equal(t, ZoomColumn, s.zoom)
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelDetail)})
	require.Equal(t, ZoomNone, s.zoom, "moving focus resets zoom")
}

func TestHiddenPanelIsNotFocusable(t *testing.T) {
	s := newTestShell("status")
	s.handleIntent(intents.ShellFocusPanel{Panel: int(PanelStatus)})
	require.Equal(t, PanelJournal, s.focus, "hidden panel ignored")
	require.False(t, s.present()[PanelStatus])
	for i := 0; i < 10; i++ {
		s.handleIntent(intents.ShellCyclePanel{Delta: 1})
		require.NotEqual(t, PanelStatus, s.focus)
	}
}

func TestJournalAndDetailCannotBeHidden(t *testing.T) {
	s := newTestShell("journal", "detail")
	require.True(t, s.present()[PanelJournal])
	require.True(t, s.present()[PanelDetail])
}

func TestToggleJournalFlipsModeAndRefreshes(t *testing.T) {
	s := newTestShell()
	before := s.journalMode.Load()
	cmd, handled := s.handleIntent(intents.ShellToggleJournal{})
	require.True(t, handled)
	require.NotNil(t, cmd)
	require.Equal(t, !before, s.journalMode.Load())
}

func TestJumpDayWithoutSourceIsConsumedButInert(t *testing.T) {
	s := newTestShell()
	cmd, handled := s.handleIntent(intents.ShellJumpDay{Forward: true})
	require.True(t, handled)
	require.Nil(t, cmd)

	s.focus = PanelDetail
	_, handled = s.handleIntent(intents.ShellJumpDay{Forward: true})
	require.False(t, handled, "day jumps belong to the Journal panel")
}

func TestScopesJournalFocused(t *testing.T) {
	s := newTestShell()
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjShell}, names(s.Scopes()))
	sc := s.Scopes()[0]
	require.Equal(t, common.LeakAll, sc.Leak, "revisions stay reachable")
	require.True(t, sc.Global, "visible behind a LeakGlobal panel scope")
}

func TestScopesDetailFocusedBlocksRevisions(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelDetail)
	scopes := s.Scopes()
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjDetail, actions.ScopeJjjShell}, names(scopes))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak)
}

func TestScopesEmptyOutsideNormalMode(t *testing.T) {
	s := newTestShell()
	s.normalMode = func() bool { return false }
	require.Empty(t, s.Scopes())
}

func TestScopesEmptyWhenShellDisabled(t *testing.T) {
	s := newTestShell()
	s.cfg.Shell.Enabled = false
	require.Empty(t, s.Scopes())
}

func TestDetailScrollIntentIsHandled(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelDetail)
	_, handled := s.handleIntent(intents.PanelScroll{Delta: 1})
	require.True(t, handled)
	_, handled = s.handleIntent(intents.PanelScroll{Delta: 1, Half: true})
	require.True(t, handled)
}

func TestSelectionEmptyUnlessAPanelProvidesOne(t *testing.T) {
	s := newTestShell()
	require.Nil(t, s.Selection().Highlighted)
}

func TestUpdateDoesNotForwardIntents(t *testing.T) {
	s := newTestShell()
	rec := &recordingPanel{}
	s.register(PanelNotes, rec)

	_, handled := s.Update(intents.Cancel{})
	require.False(t, handled)
	require.Empty(t, rec.msgs, "intents must never reach hosted models")

	_, _ = s.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	require.Empty(t, rec.msgs, "keys must never reach hosted models")

	_, _ = s.Update(common.RefreshMsg{})
	require.Len(t, rec.msgs, 1)
}

func TestCloseViewReturnsFocusToJournalFromHostedPanels(t *testing.T) {
	s := newTestShell()
	rec := &recordingPanel{}
	s.register(PanelFiles, rec)
	s.setFocus(PanelFiles)

	_, handled := s.Update(common.CloseViewMsg{})
	require.False(t, handled, "upstream still sees the close")
	require.Equal(t, PanelJournal, s.focus)
}

// recordingPanel is a minimal panel that records forwarded messages.
type recordingPanel struct{ msgs []tea.Msg }

func (r *recordingPanel) title() string                     { return "Rec" }
func (r *recordingPanel) init() tea.Cmd                     { return nil }
func (r *recordingPanel) update(msg tea.Msg) tea.Cmd        { r.msgs = append(r.msgs, msg); return nil }
func (r *recordingPanel) draw(*render.DisplayContext, layout.Box) {}
func (r *recordingPanel) scopes() []common.Scope            { return nil }
func (r *recordingPanel) setFocused(bool)                   {}
func (r *recordingPanel) editing() bool                     { return false }
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ui/shell -v`
Expected: FAIL to compile — `handleIntent`, `register`, `normalMode`, `zoom`, `Scopes` undefined.

- [ ] **Step 3: Create `panels.go`**

Create `internal/ui/shell/panels.go`:

```go
package shell

import (
	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/statuspanel"
)

// panel is one shell panel. update receives forwarded messages that are
// neither input nor intents (Shell.Update filters both); keys reach a panel
// only as intents through the scopes it returns.
type panel interface {
	title() string
	init() tea.Cmd
	update(msg tea.Msg) tea.Cmd
	draw(dl *render.DisplayContext, inner layout.Box)
	// scopes are innermost first; nil means the panel has no keys of its own.
	scopes() []common.Scope
	setFocused(focused bool)
	// editing is true while the panel owns raw input (a filter or a
	// confirmation); the shell then drops its own scope so keys reach it.
	editing() bool
}

// intentPanel is implemented by panels with jjj intents of their own.
type intentPanel interface {
	handleIntent(intent intents.Intent) (tea.Cmd, bool)
}

// selectionPanel is implemented by panels whose cursor is a selection
// (Files: the highlighted file drives Detail and $file).
type selectionPanel interface {
	selection() common.SelectionSnapshot
}

// handler is the common.ScopeHandler behind every jjj scope.
type handler struct{ s *Shell }

func (h handler) HandleIntent(intent intents.Intent) (tea.Cmd, bool) { return h.s.handleIntent(intent) }
func (h handler) Update(tea.Msg) tea.Cmd                            { return nil }

// journalPanel draws upstream's revset line and revisions list; its keys
// are upstream's own scopes, so it contributes none.
type journalPanel struct{ s *Shell }

func (p *journalPanel) title() string              { return "Journal" }
func (p *journalPanel) init() tea.Cmd              { return nil }
func (p *journalPanel) update(tea.Msg) tea.Cmd     { return nil }
func (p *journalPanel) scopes() []common.Scope     { return nil }
func (p *journalPanel) editing() bool              { return false }
func (p *journalPanel) setFocused(focused bool) {
	if p.s.hasCore {
		p.s.core.Revisions.SetFocused(focused)
	}
}
func (p *journalPanel) draw(dl *render.DisplayContext, inner layout.Box) {
	if !p.s.hasCore {
		return
	}
	rows := inner.V(layout.Fixed(1), layout.Fill(1))
	if len(rows) < 2 {
		return
	}
	p.s.core.Revset.ViewRect(dl, rows[0])
	p.s.core.RenderMain(rows[1])
}

// statusPanel hosts the three-line Status model.
type statusPanel struct {
	m *statuspanel.Model
	h handler
}

func (p *statusPanel) title() string                 { return "Status" }
func (p *statusPanel) init() tea.Cmd                 { return p.m.Init() }
func (p *statusPanel) update(msg tea.Msg) tea.Cmd    { return p.m.Update(msg) }
func (p *statusPanel) setFocused(bool)               {}
func (p *statusPanel) editing() bool                 { return false }
func (p *statusPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *statusPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjStatus, Leak: common.LeakGlobal, Handler: p.h}}
}

// detailPanel hosts the preview model; its scope scrolls it.
type detailPanel struct {
	m *preview.Model
	h handler
}

func (p *detailPanel) title() string                 { return "Detail" }
func (p *detailPanel) init() tea.Cmd                 { return p.m.Init() }
func (p *detailPanel) update(msg tea.Msg) tea.Cmd    { return p.m.Update(msg) }
func (p *detailPanel) setFocused(bool)               {}
func (p *detailPanel) editing() bool                 { return false }
func (p *detailPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *detailPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjDetail, Leak: common.LeakGlobal, Handler: p.h}}
}
func (p *detailPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	sc, ok := intent.(intents.PanelScroll)
	if !ok {
		return nil, false
	}
	switch {
	case sc.Page && sc.Delta > 0:
		return p.m.PageDown(), true
	case sc.Page:
		return p.m.PageUp(), true
	case sc.Half && sc.Delta > 0:
		return p.m.HalfPageDown(), true
	case sc.Half:
		return p.m.HalfPageUp(), true
	default:
		return p.m.Scroll(sc.Delta), true
	}
}
```

- [ ] **Step 4: Rewrite the shell around the registry**

In `internal/ui/shell/shell.go`, replace the `focusKeys` var (delete it), the `Shell` struct, `New`, `Init`, `Update`, `HandleKey`, `setFocus`, `present`, `Render` and `panel` with the following (`Attach` and `rowSource` stay as they are; add imports `"github.com/idursun/jjui/internal/ui/intents"`):

```go
// Shell is the ui.Extension that lays out jjj's panels and routes keys to
// them through the scopes it contributes (ui.go's ScopeProvider seam).
type Shell struct {
	ctx     *appContext.MainContext
	cfg     *jjj.Config
	core    ui.Core
	hasCore bool

	panels map[PanelID]panel
	focus  PanelID
	zoom   Zoom
	detail *preview.Model // also driven directly by Files (annotate)
	h      handler
	// normalMode reports whether no upstream operation, modal or quick
	// search is open on the revisions list; tests replace it.
	normalMode func() bool
	// detailFile, while Detail is focused, keeps Detail on a file diff
	// opened from Files (enter) instead of the selected revision.
	detailFile *common.SelectedFile

	journalMode atomic.Bool                    // read from the load goroutine
	source      atomic.Pointer[journal.Source] // last journal source, for jump keys
	dayStyle    lipgloss.Style                 // header styles, resolved on the UI loop in Attach
	weekStyle   lipgloss.Style
}

var (
	_ common.ScopeProvider     = (*Shell)(nil)
	_ common.SelectionProvider = (*Shell)(nil)
)

// panelNames maps [shell] hidden entries to panels. Journal and Detail
// cannot be hidden.
var panelNames = map[string]PanelID{
	"status": PanelStatus, "files": PanelFiles, "bookmarks": PanelBookmarks, "notes": PanelNotes,
}

func New(ctx *appContext.MainContext, cfg *jjj.Config) *Shell {
	s := &Shell{ctx: ctx, cfg: cfg, panels: map[PanelID]panel{}, focus: PanelJournal, detail: preview.New(ctx)}
	s.h = handler{s}
	s.normalMode = func() bool {
		return s.hasCore && s.core.Revisions.InNormalMode() && !s.core.Revisions.HasQuickSearch()
	}
	s.journalMode.Store(cfg.Journal.Enabled)
	status := statuspanel.New(ctx, func() string {
		if s.journalMode.Load() {
			return "journal"
		}
		return "graph"
	})
	s.register(PanelStatus, &statusPanel{m: status, h: s.h})
	s.register(PanelJournal, &journalPanel{s: s})
	s.register(PanelDetail, &detailPanel{m: s.detail, h: s.h})
	return s
}

func (s *Shell) register(id PanelID, p panel) { s.panels[id] = p }

func (s *Shell) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, id := range Order {
		if p, ok := s.panels[id]; ok {
			cmds = append(cmds, p.init())
		}
	}
	return tea.Batch(cmds...)
}

// present is the set of registered, non-hidden panels.
func (s *Shell) present() map[PanelID]bool {
	out := make(map[PanelID]bool, len(s.panels))
	for id := range s.panels {
		out[id] = true
	}
	for _, name := range s.cfg.Shell.Hidden {
		if id, ok := panelNames[name]; ok {
			delete(out, id)
		}
	}
	return out
}

func (s *Shell) setFocus(id PanelID) {
	if !s.present()[id] {
		return
	}
	if id != s.focus {
		s.zoom = ZoomNone
	}
	if id != PanelDetail {
		s.detailFile = nil
	}
	s.focus = id
	for pid, p := range s.panels {
		p.setFocused(pid == id)
	}
}

// Scopes: the focused panel's scopes (innermost first), then jjj.shell,
// which is LeakAll (revisions stay reachable while Journal is focused) and
// Global (still visible behind a LeakGlobal panel scope). Nothing while an
// upstream operation is open or a panel owns raw input — except that panel's
// own scopes.
func (s *Shell) Scopes() []common.Scope {
	if !s.cfg.Shell.Enabled || !s.normalMode() {
		return nil
	}
	p := s.panels[s.focus]
	var scopes []common.Scope
	if p != nil {
		scopes = append(scopes, p.scopes()...)
		if p.editing() {
			return scopes
		}
	}
	return append(scopes, common.Scope{Name: actions.ScopeJjjShell, Leak: common.LeakAll, Global: true, Handler: s.h})
}

// Selection: a file opened into Detail, else the focused panel's selection.
func (s *Shell) Selection() common.SelectionSnapshot {
	if s.focus == PanelDetail && s.detailFile != nil {
		return common.SelectionSnapshot{Highlighted: *s.detailFile}
	}
	if sp, ok := s.panels[s.focus].(selectionPanel); ok {
		return sp.selection()
	}
	return common.SelectionSnapshot{}
}

func (s *Shell) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch it := intent.(type) {
	case intents.ShellFocusPanel:
		s.setFocus(PanelID(it.Panel))
		return nil, true
	case intents.ShellCyclePanel:
		s.setFocus(Cycle(s.present(), s.focus, it.Delta))
		return nil, true
	case intents.ShellZoom:
		s.zoom = s.zoom.Step(it.Delta)
		return nil, true
	case intents.ShellToggleJournal:
		s.journalMode.Store(!s.journalMode.Load())
		s.source.Store(nil) // drop the stale source: {/} must not use old header indices during the reload
		return common.RefreshAndKeepSelections, true
	case intents.ShellJumpDay:
		if s.focus != PanelJournal {
			return nil, false
		}
		if src := s.source.Load(); src != nil && s.hasCore {
			if idx := journal.JumpTarget(src.HeaderIndices(), s.core.Revisions.Cursor(), it.Forward); idx >= 0 {
				s.core.Revisions.SetCursor(idx)
			}
		}
		return nil, true
	}
	if ip, ok := s.panels[s.focus].(intentPanel); ok {
		return ip.handleIntent(intent)
	}
	return nil, false
}

// Update handles panel clicks and focus-returning closes, then forwards
// every message that is neither input nor an intent to all panels. Intents
// travelling as messages must not reach hosted upstream models, which would
// act on them (a Cancel closing a confirmation, say).
func (s *Shell) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg.(type) {
	case focusPanelMsg:
		s.setFocus(msg.(focusPanelMsg).id)
		return nil, true
	case revisions.PaneClickedMsg, revisions.ItemClickedMsg:
		s.setFocus(PanelJournal) // let revisions handle the click too
		return nil, false
	case common.CloseViewMsg:
		// esc in Files or Bookmarks (upstream emits Close) returns to Journal.
		if s.focus == PanelFiles || s.focus == PanelBookmarks {
			s.setFocus(PanelJournal)
		}
		return nil, false
	}
	if common.IsInputMessage(msg) {
		return nil, false
	}
	if _, ok := msg.(intents.Intent); ok {
		return nil, false
	}
	var cmds []tea.Cmd
	for _, id := range Order {
		if p, ok := s.panels[id]; ok {
			cmds = append(cmds, p.update(msg))
		}
	}
	return tea.Batch(cmds...), false
}

// HandleKey is inert: every jjj key is an action in a jjj scope (Scopes),
// so it is rebindable and routed by upstream's dispatcher.
func (s *Shell) HandleKey(tea.KeyMsg) (tea.Cmd, bool) { return nil, false }

// Render draws the panels. Below [shell] narrow_columns it draws the stock
// layout while Journal is focused, and the focused panel full-screen
// otherwise (digits still focus panels there).
func (s *Shell) Render(dl *render.DisplayContext, box layout.Box, core ui.Core) bool {
	s.core, s.hasCore = core, true
	if !s.cfg.Shell.Enabled {
		return false
	}
	content, statusLine := box.CutBottom(1)
	if box.R.Dx() < s.cfg.Shell.NarrowColumns {
		if s.focus == PanelJournal {
			return false
		}
		s.drawPanel(dl, content, s.focus)
		core.Status.ViewRect(dl, statusLine)
		return true
	}
	lay := Compute(content, Params{
		LeftWidth: s.cfg.Shell.LeftWidth,
		NotesRows: s.cfg.Shell.NotesRows,
		Focus:     s.focus,
		Zoom:      s.zoom,
		Present:   s.present(),
	})
	for _, id := range Order {
		if b, ok := lay[id]; ok {
			s.drawPanel(dl, b, id)
		}
	}
	core.Status.ViewRect(dl, statusLine)
	return true
}

// drawPanel draws a rounded border with a "[n]─Title" tab, registers a click
// target that focuses the panel, then draws the content inside the border.
func (s *Shell) drawPanel(dl *render.DisplayContext, box layout.Box, id PanelID) {
	p := s.panels[id]
	w, h := box.R.Dx(), box.R.Dy()
	if p == nil || w < 4 || h < 3 {
		return
	}
	style := borderUnfocused
	if s.focus == id {
		style = borderFocused
	}
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.GetForeground()).
		Width(w - 2).
		Height(h - 2).
		Render("")
	dl.AddDraw(box.R, frame, render.ZBase)
	dl.AddInteraction(box.R, focusPanelMsg{id: id}, render.InteractionClick, -1)
	dl.Text(box.R.Min.X+2, box.R.Min.Y, render.ZBase).
		Styled(fmt.Sprintf("[%d]─%s", int(id), p.title()), style.Bold(s.focus == id)).
		Done()
	p.draw(dl, box.Inset(1))
}
```

Remove the now-unused `status` and `active` fields and the old `panel` method. Keep `focusPanelMsg`, `borderFocused`, `borderUnfocused`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/ui/shell -v`
Expected: PASS (all tests in `shell_test.go` and `layout_test.go`).

- [ ] **Step 6: Smoke the binary**

Run: `make smoke REPO=$HOME/Developer/Home/ticktock`
Expected: exit 0 (the smoke expects `revset:`, drawn inside the Journal panel).

- [ ] **Step 7: Verify and commit**

Run: `make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): shell panel registry, scoped keys, narrow full-screen panels (TASK-012.01, TASK-006)"
```

---

### Task 5: Notes panel (6)

**Files:**
- Create: `internal/ui/notes/notes.go`, `internal/ui/notes/notes_test.go`
- Modify: `internal/ui/shell/panels.go` (add `notesPanel`), `internal/ui/shell/shell.go` (`New` registers it; `handleIntent` gains no cases — `NotesDescribe` is handled by the panel)

**Interfaces:**
- Produces (package `notes`): `func New(ctx *appContext.MainContext) *Model`; `func (m *Model) Update(msg tea.Msg) tea.Cmd`; `func (m *Model) Lines(width int) []string`; `func (m *Model) Scroll(delta int)`; `func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box)`; `func (m *Model) ChangeID() string`.
- Consumes: `common.SelectionChangedMsg` with a `common.SelectedRevision` item (file selections are ignored), `common.UpdateRevisionsSuccessMsg`; `jj.GetDescription`.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/notes/notes_test.go`:

```go
package notes

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/test"
)

func TestLinesNoDescription(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.text, m.loaded = "  \n", true
	require.Equal(t, []string{"(no description)"}, m.Lines(40))
}

func TestLinesWrapsToWidth(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.text, m.loaded = "feat: subject\n\nthe body is long enough to wrap\n", true
	lines := m.Lines(12)
	require.Equal(t, "feat:", lines[0][:5])
	for _, l := range lines {
		require.LessOrEqual(t, len(l), 12, "line %q", l)
	}
}

func TestSelectionChangeLoadsDescription(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.GetDescription("abc")).SetOutput([]byte("subject\n"))
	defer runner.Verify()
	m := New(test.NewTestContext(runner))

	cmd := m.Update(common.SelectionChangedMsg{Item: common.SelectedRevision{ChangeId: "abc", CommitId: "1"}})
	require.NotNil(t, cmd)
	m.Update(drainDebounced(t, cmd))

	require.Equal(t, "abc", m.ChangeID())
	require.Equal(t, []string{"subject"}, m.Lines(40))
}

func TestFileSelectionKeepsRevision(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.changeID = "abc"
	cmd := m.Update(common.SelectionChangedMsg{Item: common.SelectedFile{ChangeId: "abc", File: jj.NewFileName("a")}})
	require.Nil(t, cmd)
	require.Equal(t, "abc", m.ChangeID())
}

func TestStaleDescriptionIgnored(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	m.seq = 2
	m.Update(descMsg{changeID: "old", text: "stale", seq: 1})
	require.False(t, m.loaded)
}

// drainDebounced runs a (debounced) command and returns its message.
func drainDebounced(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	return cmd()
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/ui/notes -v`
Expected: FAIL to compile — package `notes` has no `New`.

- [ ] **Step 3: Implement the model**

Create `internal/ui/notes/notes.go`:

```go
// Package notes is jjj's Notes panel (spec §4.4): the selected revision's
// full description, wrapped, first line bold.
package notes

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	debounceID       = "jjj.notes"
	debounceDuration = 50 * time.Millisecond
)

type descMsg struct {
	changeID, text string
	seq            uint64
}

type Model struct {
	ctx      *appContext.MainContext
	changeID string
	text     string
	loaded   bool
	seq      uint64
	offset   int
}

func New(ctx *appContext.MainContext) *Model { return &Model{ctx: ctx} }

func (m *Model) ChangeID() string { return m.changeID }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		rev, ok := msg.Item.(common.SelectedRevision)
		if !ok || rev.ChangeId == m.changeID {
			return nil
		}
		m.changeID, m.offset = rev.ChangeId, 0
		return m.reload()
	case common.UpdateRevisionsSuccessMsg:
		if m.changeID != "" {
			return m.reload()
		}
	case descMsg:
		if msg.seq == m.seq && msg.changeID == m.changeID {
			m.text, m.loaded = msg.text, true
		}
	}
	return nil
}

func (m *Model) reload() tea.Cmd {
	m.seq++
	seq, id, ctx := m.seq, m.changeID, m.ctx
	return common.Debounce(debounceID, debounceDuration, func() tea.Msg {
		out, err := ctx.RunCommandImmediate(jj.GetDescription(id))
		if err != nil {
			return descMsg{changeID: id, seq: seq}
		}
		return descMsg{changeID: id, text: string(out), seq: seq}
	})
}

// Lines is the description wrapped to width; "(no description)" when empty.
func (m *Model) Lines(width int) []string {
	text := strings.TrimRight(m.text, "\n ")
	if strings.TrimSpace(text) == "" {
		return []string{"(no description)"}
	}
	return strings.Split(ansi.Wrap(text, max(width, 1), ""), "\n")
}

func (m *Model) Scroll(delta int) { m.offset = max(0, m.offset+delta) }

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if !m.loaded {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Write("…").Done()
		return
	}
	lines := m.Lines(box.R.Dx())
	m.offset = min(m.offset, max(0, len(lines)-1))
	bold := lipgloss.NewStyle().Bold(true)
	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase)
	for i, line := range lines[m.offset:] {
		if i >= box.R.Dy() {
			break
		}
		if i > 0 {
			tb.NewLine()
		}
		if i+m.offset == 0 {
			tb.Styled(line, bold)
		} else {
			tb.Write(line)
		}
	}
	tb.Done()
}
```

- [ ] **Step 4: Run the notes tests**

Run: `go test ./internal/ui/notes -v`
Expected: PASS. (If `TestSelectionChangeLoadsDescription` hangs on the debounce, `common.Debounce` sleeps its duration before calling the inner func; the 50 ms wait is expected.)

- [ ] **Step 5: Register the panel**

In `internal/ui/shell/panels.go` add (import `"github.com/idursun/jjui/internal/ui/notes"`):

```go
// notesPanel hosts the Notes model; enter opens inline describe on Journal.
type notesPanel struct {
	s *Shell
	m *notes.Model
}

func (p *notesPanel) title() string                 { return "Notes" }
func (p *notesPanel) init() tea.Cmd                 { return nil }
func (p *notesPanel) update(msg tea.Msg) tea.Cmd    { return p.m.Update(msg) }
func (p *notesPanel) setFocused(bool)               {}
func (p *notesPanel) editing() bool                 { return false }
func (p *notesPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *notesPanel) scopes() []common.Scope {
	return []common.Scope{{Name: actions.ScopeJjjNotes, Leak: common.LeakGlobal, Handler: p.s.h}}
}
func (p *notesPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch it := intent.(type) {
	case intents.PanelScroll:
		p.m.Scroll(it.Delta)
		return nil, true
	case intents.NotesDescribe:
		if !p.s.hasCore {
			return nil, true
		}
		rev := p.s.core.Revisions.SelectedRevision()
		if rev == nil {
			return nil, true
		}
		p.s.setFocus(PanelJournal)
		cmd, _ := p.s.core.Revisions.HandleIntent(intents.OpenInlineDescribe{Selected: rev})
		return cmd, true
	}
	return nil, false
}
```

In `New` (shell.go), after registering Detail:

```go
	s.register(PanelNotes, &notesPanel{s: s, m: notes.New(ctx)})
```

Add to `shell_test.go`:

```go
func TestNotesRegisteredAndScoped(t *testing.T) {
	s := newTestShell()
	require.True(t, s.present()[PanelNotes])
	s.setFocus(PanelNotes)
	require.Equal(t, []keybindings.ScopeName{actions.ScopeJjjNotes, actions.ScopeJjjShell}, names(s.Scopes()))
	_, handled := s.handleIntent(intents.PanelScroll{Delta: 1})
	require.True(t, handled)
	_, handled = s.handleIntent(intents.NotesDescribe{})
	require.True(t, handled, "consumed even without a core")
}
```

- [ ] **Step 6: Verify and commit**

Run: `go test ./internal/ui/shell ./internal/ui/notes -v && make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): Notes panel — full description, wrap, enter to describe (TASK-012.01)"
```

---

### Task 6: Files panel (2)

**Files:**
- Create: `internal/jj/jjj_commands.go`, `internal/jj/jjj_commands_test.go`
- Create: `internal/ui/files/files.go`, `internal/ui/files/files_test.go`
- Modify: `internal/ui/shell/panels.go` (add `filesPanel`), `internal/ui/shell/shell.go` (`New` registers it; `Update` handles `details.FileClickedMsg` and `files.DetailContentMsg`; `handleIntent` handles `FilesShowDiff`)
- Modify: `UPSTREAM.md`

**Interfaces:**
- Produces (package `jj`, new file): `func BookmarkAdvance(to string, names ...string) CommandArgs`, `func FileAnnotate(revision string, file FileName) CommandArgs`, `func ResolveFiles(revision string, files []FileName) CommandArgs`, `func DiffEditFiles(revision string, files []FileName) CommandArgs`.
- Produces (package `files`): `func New(ctx *appContext.MainContext, isWorkingCopy func() bool) *Model`; `Update(msg tea.Msg) tea.Cmd`; `Scopes(h common.ScopeHandler) []common.Scope`; `IsEditing() bool`; `Selection() common.SelectionSnapshot`; `Current() (common.SelectedFile, bool)`; `HandleIntent(intents.Intent) (tea.Cmd, bool)`; `ViewRect(dl, box)`; `type DetailContentMsg struct{ Content string }`.
- Consumes: `details.NewOperation`, `(*details.Operation).Init/Update/Scopes/IsEditing/Selection/ViewRect`.

- [ ] **Step 1: Write the failing `jj` tests**

Create `internal/jj/jjj_commands_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify failure, then implement**

Run: `go test ./internal/jj -run 'TestBookmarkAdvance|TestFileCommands' -v` — Expected: FAIL to compile.

Create `internal/jj/jjj_commands.go`:

```go
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
```

Run: `go test ./internal/jj -run 'TestBookmarkAdvance|TestFileCommands' -v` — Expected: PASS.

- [ ] **Step 3: Write the failing Files tests**

Create `internal/ui/files/files_test.go`:

```go
package files

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/test"
)

type nopHandler struct{}

func (nopHandler) HandleIntent(intents.Intent) (tea.Cmd, bool) { return nil, false }
func (nopHandler) Update(tea.Msg) tea.Cmd                      { return nil }

var revA = common.SelectedRevision{ChangeId: "aaa", CommitId: "1"}
var revB = common.SelectedRevision{ChangeId: "bbb", CommitId: "2"}

func TestFilesIgnoresStaleRevision(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })

	require.NotNil(t, m.Update(common.SelectionChangedMsg{Item: revA}))
	require.NotNil(t, m.Update(common.SelectionChangedMsg{Item: revB}))
	// The debounced message for A arrives after B was requested: no load
	// (the test runner would fail on an unexpected jj command).
	require.Nil(t, m.Update(revisionMsg{rev: revA}))
	require.Nil(t, m.op, "nothing loaded for a superseded revision")
}

func TestFilesSameRevisionIsNoop(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	m.Update(common.SelectionChangedMsg{Item: revA})
	require.Nil(t, m.Update(common.SelectionChangedMsg{Item: revA}))
}

func TestFilesIgnoresFileSelections(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	require.Nil(t, m.Update(common.SelectionChangedMsg{Item: common.SelectedFile{ChangeId: "aaa", File: jj.NewFileName("x")}}))
}

func TestFilesLoadsOnLatestRevision(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.Snapshot())
	runner.Expect(jj.Status("aaa")).SetOutput([]byte("false $\nM a.go\n"))
	defer runner.Verify()
	m := New(test.NewTestContext(runner), func() bool { return true })

	m.Update(common.SelectionChangedMsg{Item: revA})
	cmd := m.Update(revisionMsg{rev: revA})
	require.NotNil(t, m.op)
	if cmd != nil {
		m.Update(cmd())
	}
}

func TestScopesWithoutOp(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	scopes := m.Scopes(nopHandler{})
	require.Len(t, scopes, 1)
	require.Equal(t, actions.ScopeJjjFiles, string(scopes[0].Name))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak, "never leak to revisions, even before loading")
}

func TestFileIntentsWithoutFilesAreConsumed(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)), func() bool { return true })
	for _, it := range []intents.Intent{intents.FilesDiffEdit{}, intents.FilesOpenEditor{}, intents.FilesAnnotate{}, intents.FilesResolve{}} {
		cmd, handled := m.HandleIntent(it)
		require.True(t, handled, "%T", it)
		require.Nil(t, cmd, "%T", it)
	}
}
```

The status output format above (`false $` line, then `M a.go`) mirrors what `details.createListItems` parses (conflict flags, `$`, then the file lines). If `jj.Status` produces a different summary format at this pin, copy a real sample from `internal/ui/operations/details/details_test.go`.

- [ ] **Step 4: Run to verify failure**

Run: `go test ./internal/ui/files -v`
Expected: FAIL to compile — package `files` has no `New`.

- [ ] **Step 5: Implement the Files model**

Create `internal/ui/files/files.go`:

```go
// Package files is jjj's Files panel (spec §4.2): the files changed in the
// Journal-selected revision, hosted from upstream's file-details operation
// so its keys (space r s S A d / *) work unchanged, plus jjj file actions.
package files

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/operations/details"
	"github.com/idursun/jjui/internal/ui/render"
)

const (
	debounceID = "jjj.files"
	// debounceDuration: details.Operation loads synchronously (jj snapshot +
	// status) on the UI loop, so wait for the cursor to settle.
	debounceDuration = 120 * time.Millisecond
)

// revisionMsg is the debounced "load this revision" request.
type revisionMsg struct{ rev common.SelectedRevision }

// DetailContentMsg asks the shell to show Content in Detail (annotate).
type DetailContentMsg struct{ Content string }

type Model struct {
	ctx           *appContext.MainContext
	isWorkingCopy func() bool // is the Journal-selected revision @?
	op            *details.Operation
	rev           common.SelectedRevision // latest requested
}

func New(ctx *appContext.MainContext, isWorkingCopy func() bool) *Model {
	return &Model{ctx: ctx, isWorkingCopy: isWorkingCopy}
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case common.SelectionChangedMsg:
		rev, ok := msg.Item.(common.SelectedRevision)
		if !ok || rev == m.rev {
			return nil
		}
		m.rev = rev
		return common.Debounce(debounceID, debounceDuration, func() tea.Msg { return revisionMsg{rev: rev} })
	case revisionMsg:
		if msg.rev != m.rev {
			return nil
		}
		commit := &jj.Commit{ChangeId: msg.rev.ChangeId, CommitId: msg.rev.CommitId}
		if m.op == nil {
			m.op = details.NewOperation(m.ctx, commit)
			return m.op.Init()
		}
		return m.op.Update(common.SelectionChangedMsg{Item: msg.rev})
	}
	if m.op == nil {
		return nil
	}
	return m.op.Update(msg)
}

// Scopes: jjj.files first (LeakAll, so the details scopes behind it stay
// reachable), then upstream's details scopes (LeakGlobal: revisions are
// blocked). While the op owns raw input only its scopes are returned.
func (m *Model) Scopes(h common.ScopeHandler) []common.Scope {
	if m.op == nil {
		return []common.Scope{{Name: actions.ScopeJjjFiles, Leak: common.LeakGlobal, Handler: h}}
	}
	if m.op.IsEditing() {
		return m.op.Scopes()
	}
	return append([]common.Scope{{Name: actions.ScopeJjjFiles, Leak: common.LeakAll, Handler: h}}, m.op.Scopes()...)
}

func (m *Model) IsEditing() bool { return m.op != nil && m.op.IsEditing() }

// Selection reports only the highlighted file; checked files stay private
// so they never mix with checked revisions in the shared context.
func (m *Model) Selection() common.SelectionSnapshot {
	if m.op == nil {
		return common.SelectionSnapshot{}
	}
	return common.SelectionSnapshot{Highlighted: m.op.Selection().Highlighted}
}

func (m *Model) Current() (common.SelectedFile, bool) {
	if m.op == nil {
		return common.SelectedFile{}, false
	}
	f, ok := m.op.Selection().Highlighted.(common.SelectedFile)
	return f, ok
}

// targets: the checked files, or the highlighted one when none is checked.
func (m *Model) targets() (string, []jj.FileName) {
	cur, ok := m.Current()
	if !ok {
		return "", nil
	}
	var files []jj.FileName
	for _, c := range m.op.Selection().Checked {
		if f, ok := c.(common.SelectedFile); ok {
			files = append(files, f.File)
		}
	}
	if len(files) == 0 {
		files = []jj.FileName{cur.File}
	}
	return cur.ChangeId, files
}

func (m *Model) HandleIntent(intent intents.Intent) (tea.Cmd, bool) {
	switch intent.(type) {
	case intents.FilesDiffEdit:
		rev, files := m.targets()
		if rev == "" {
			return nil, true
		}
		return m.ctx.RunInteractiveCommand(jj.DiffEditFiles(rev, files), common.Refresh), true
	case intents.FilesResolve:
		rev, files := m.targets()
		if rev == "" {
			return nil, true
		}
		return m.ctx.RunInteractiveCommand(jj.ResolveFiles(rev, files), common.Refresh), true
	case intents.FilesAnnotate:
		cur, ok := m.Current()
		if !ok {
			return nil, true
		}
		ctx := m.ctx
		return func() tea.Msg {
			out, err := ctx.RunCommandImmediate(jj.FileAnnotate(cur.ChangeId, cur.File))
			if err != nil {
				return common.CommandCompletedMsg{Output: string(out), Err: err}
			}
			return DetailContentMsg{Content: string(out)}
		}, true
	case intents.FilesOpenEditor:
		cur, ok := m.Current()
		if !ok {
			return nil, true
		}
		if !m.isWorkingCopy() {
			return intents.Invoke(intents.AddMessage{Text: "open in $EDITOR works on the working copy (@) only"}), true
		}
		line := "${EDITOR:-vi} " + cur.File.ShellEscaped()
		return func() tea.Msg { return common.ExecMsg{Line: line, Mode: common.ExecShell} }, true
	}
	return nil, false
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	if m.op == nil {
		dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase).Write("…").Done()
		return
	}
	m.op.ViewRect(dl, box)
}
```

Check before running: `intents.AddMessage` exists (used by `bookmarkpane/actions.go`), `common.ExecMsg` has fields `Line` and `Mode` (`internal/ui/common/msgs.go:43`), and `jj.Snapshot` / `jj.Status` exist (used by `details.load`).

- [ ] **Step 6: Run the Files tests**

Run: `go test ./internal/ui/files -v`
Expected: PASS.

- [ ] **Step 7: Wire the panel into the shell**

In `internal/ui/shell/panels.go` add (imports `"github.com/idursun/jjui/internal/ui/files"`):

```go
type filesPanel struct {
	s *Shell
	m *files.Model
}

func (p *filesPanel) title() string                 { return "Files" }
func (p *filesPanel) init() tea.Cmd                 { return nil }
func (p *filesPanel) update(msg tea.Msg) tea.Cmd    { return p.m.Update(msg) }
func (p *filesPanel) setFocused(bool)               {}
func (p *filesPanel) editing() bool                 { return p.m.IsEditing() }
func (p *filesPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *filesPanel) scopes() []common.Scope        { return p.m.Scopes(p.s.h) }
func (p *filesPanel) selection() common.SelectionSnapshot { return p.m.Selection() }
func (p *filesPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	return p.m.HandleIntent(intent)
}
```

In `New` (shell.go), after registering Notes:

```go
	s.register(PanelFiles, &filesPanel{s: s, m: files.New(ctx, func() bool {
		if !s.hasCore {
			return false
		}
		rev := s.core.Revisions.SelectedRevision()
		return rev != nil && rev.IsWorkingCopy
	})})
```

In `handleIntent`, add a case before the `intentPanel` fallback (the shell owns focus, so it handles "open into Detail"):

```go
	case intents.FilesShowDiff:
		fp, ok := s.panels[PanelFiles].(*filesPanel)
		if !ok {
			return nil, true
		}
		if f, ok := fp.m.Current(); ok {
			s.setFocus(PanelDetail)
			s.detailFile = &f
		}
		return nil, true
```

In `Update`, add cases to the type switch (import `"github.com/idursun/jjui/internal/ui/operations/details"`):

```go
	case details.FileClickedMsg:
		s.setFocus(PanelFiles)
		return s.panels[PanelFiles].update(msg), true
	case files.DetailContentMsg:
		s.detail.SetContent(msg.(files.DetailContentMsg).Content)
		return nil, true
```

Add to `shell_test.go`:

```go
func TestScopesDropShellWhileEditing(t *testing.T) {
	s := newTestShell()
	rec := &editingPanel{}
	s.register(PanelFiles, rec)
	s.setFocus(PanelFiles)
	rec.isEditing = true
	for _, sc := range s.Scopes() {
		require.NotEqual(t, actions.ScopeJjjShell, string(sc.Name), "shell keys must not steal input")
	}
}

func TestFilesFocusedBlocksRevisions(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelFiles)
	scopes := s.Scopes()
	require.Equal(t, actions.ScopeJjjFiles, string(scopes[0].Name))
	require.Equal(t, common.LeakGlobal, scopes[0].Leak)
	require.Equal(t, actions.ScopeJjjShell, string(scopes[len(scopes)-1].Name))
}

type editingPanel struct {
	recordingPanel
	isEditing bool
}

func (e *editingPanel) editing() bool { return e.isEditing }
func (e *editingPanel) scopes() []common.Scope {
	return []common.Scope{{Name: "fake.filter", Leak: common.LeakNone, Handler: handler{}}}
}
```

- [ ] **Step 8: Record and commit**

Add to `UPSTREAM.md`'s table:

```markdown
| `internal/jj/jjj_commands.go` | new file: jjj command builders (`BookmarkAdvance`, `FileAnnotate`, `ResolveFiles`, `DiffEditFiles`) |
```

Run: `go test ./internal/ui/shell ./internal/ui/files ./internal/jj -v && make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): Files panel — live file list, diff in Detail, diffedit/annotate/resolve/editor (TASK-012.01)"
```

---

### Task 7: Bookmarks panel (3) with advance

**Files:**
- Create: `internal/ui/bookmarkpane/advance.go`, `internal/ui/bookmarkpane/advance_test.go`
- Create: `internal/ui/shell/bookmarks.go`
- Modify: `internal/ui/shell/shell.go` (`New` registers it; `Update` handles the pane's click and outbound messages)
- Modify: `UPSTREAM.md`

**Interfaces:**
- Consumes: `jj.BookmarkAdvance` (Task 6), `jj.GitPush`, the pane's private `currentCommitID`, `selected`, `selectionsForBookmarkOperation`, `openConfirmation`, `context`.
- Produces: `func (m *bookmarkpane.Model) Advance(toWorkingCopy bool) tea.Cmd`; shell `bookmarksPanel` with scopes `[jjj.bookmarks (LeakAll), bookmark_pane…]`.

- [ ] **Step 1: Write the failing pane tests**

Create `internal/ui/bookmarkpane/advance_test.go`:

```go
package bookmarkpane

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/test"
)

func newAdvancePane(t *testing.T, runner *test.CommandRunner) *Model {
	t.Helper()
	m := New(test.NewTestContext(runner))
	m.currentCommitID = "abc123"
	return m
}

func TestAdvanceClosestToSelectedRevision(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkAdvance("abc123"))
	defer runner.Verify()
	m := newAdvancePane(t, runner)

	cmd := m.Advance(false)
	require.NotNil(t, cmd)
	test.SimulateModel(m, cmd)
}

func TestAdvanceSelectedLocalBookmarksByName(t *testing.T) {
	runner := test.NewTestCommandRunner(t)
	runner.Expect(jj.BookmarkListAll()).SetOutput([]byte("main;.;true;false;false;false;abc123\nfeat;.;true;false;false;false;def456\n"))
	runner.Expect(jj.BookmarkAdvance("abc123", "feat"))
	defer runner.Verify()
	m := newAdvancePane(t, runner)
	test.SimulateModel(m, m.loadRows)
	m.selected["feat"] = true

	test.SimulateModel(m, m.Advance(false))
}

func TestAdvanceWithoutRevisionContextDoesNothing(t *testing.T) {
	m := New(test.NewTestContext(test.NewTestCommandRunner(t)))
	require.Nil(t, m.Advance(false))
}

func TestAdvanceToWorkingCopyAsksBeforePushing(t *testing.T) {
	m := newAdvancePane(t, test.NewTestCommandRunner(t))
	cmd := m.Advance(true)
	require.NotNil(t, cmd)
	require.NotNil(t, m.confirmation, "advance + push waits for a yes")
}
```

Check before running: the line format `name;remote;present;…;commit` must match `jj.BookmarkListAll`'s template at this pin — copy a real line from `internal/ui/bookmarkpane_integration_test.go` (it uses `main;.;true;false;false;false;abc123`) and from `bookmarkpane_test.go` if the field count differs. `test.SimulateModel` takes a model with `Update(tea.Msg) tea.Cmd`; `*Model` satisfies it.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/ui/bookmarkpane -run TestAdvance -v`
Expected: FAIL to compile — `m.Advance` undefined.

- [ ] **Step 3: Implement `Advance`**

Create `internal/ui/bookmarkpane/advance.go`:

```go
package bookmarkpane

// jjj addition (new file; upstream PR candidate — see UPSTREAM.md).

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/ui/common"
)

// Advance runs `jj bookmark advance` toward the revision selected in the
// revisions list, or toward @ when toWorkingCopy is set. Multi-selected local
// bookmarks are passed by exact name; with none selected jj advances the
// closest bookmarks. toWorkingCopy also pushes, after one confirmation:
// the selected bookmarks by name, or jj's default push set when none are
// selected (bookmarks between their remote and @).
func (m *Model) Advance(toWorkingCopy bool) tea.Cmd {
	to := m.currentCommitID
	if toWorkingCopy {
		to = "@"
	}
	if to == "" {
		return nil
	}
	names := m.advanceNames()
	advance := jj.BookmarkAdvance(to, names...)
	if !toWorkingCopy {
		return m.context.RunCommand(advance, common.Refresh)
	}
	var flags []string
	for _, n := range names {
		flags = append(flags, "--bookmark", n)
	}
	what := "the closest bookmarks"
	if len(names) > 0 {
		what = strings.Join(names, ", ")
	}
	return m.openConfirmation(
		[]string{fmt.Sprintf("Advance %s to @ and push?", what)},
		tea.Sequence(m.context.RunCommand(advance), m.context.RunCommand(jj.GitPush(flags...), common.Refresh)),
	)
}

// advanceNames are the multi-selected local bookmark names, in tree order.
func (m *Model) advanceNames() []string {
	if len(m.selected) == 0 {
		return nil
	}
	var names []string
	seen := map[string]bool{}
	for _, s := range m.selectionsForBookmarkOperation() {
		if s.node.IsRemote() || seen[s.bookmark.Name] {
			continue
		}
		seen[s.bookmark.Name] = true
		names = append(names, s.bookmark.Name)
	}
	return names
}
```

Check: `openConfirmation(messages []string, cmd tea.Cmd) tea.Cmd` is defined at `internal/ui/bookmarkpane/actions.go:312` and sets `m.confirmation`; if its signature differs at this pin, adapt the call, not the upstream function.

- [ ] **Step 4: Run the pane tests**

Run: `go test ./internal/ui/bookmarkpane -v`
Expected: PASS (new and existing tests).

- [ ] **Step 5: Write the failing shell tests**

Add to `shell_test.go`:

```go
func TestBookmarksFocusedScopes(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	got := names(s.Scopes())
	require.Equal(t, actions.ScopeJjjBookmarks, string(got[0]))
	require.Equal(t, actions.ScopeBookmarkPane, string(got[1]))
	require.Equal(t, actions.ScopeJjjShell, string(got[len(got)-1]))
}

func TestBookmarksScopesWhileEditing(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	bp := s.panels[PanelBookmarks].(*bookmarksPanel)
	bp.m.HandleIntent(intents.BookmarkPaneOpenFilter{})
	got := names(s.Scopes())
	require.Equal(t, []keybindings.ScopeName{actions.ScopeBookmarkPaneFilter}, got)
}

func TestAdvanceIntentReachesPane(t *testing.T) {
	s := newTestShell()
	s.setFocus(PanelBookmarks)
	_, handled := s.handleIntent(intents.BookmarkPaneAdvance{})
	require.True(t, handled)
}
```

Run: `go test ./internal/ui/shell -run 'TestBookmarks|TestAdvanceIntent' -v` — Expected: FAIL (no Bookmarks panel registered).

- [ ] **Step 6: Implement the panel**

Create `internal/ui/shell/bookmarks.go`:

```go
package shell

import (
	tea "charm.land/bubbletea/v2"

	"github.com/idursun/jjui/internal/ui/actions"
	"github.com/idursun/jjui/internal/ui/bookmarkpane"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/intents"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

// bookmarksPanel hosts upstream's bookmark pane persistently (spec §4.3).
type bookmarksPanel struct {
	s *Shell
	m *bookmarkpane.Model
}

func (p *bookmarksPanel) title() string { return "Bookmarks" }
func (p *bookmarksPanel) init() tea.Cmd { return p.m.OnShow() }

// update keeps the pane's revision context in sync, as ui.syncBookmarkPaneContext
// does for the split pane: it sorts by proximity and advances toward it.
func (p *bookmarksPanel) update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case common.SelectionChangedMsg, common.UpdateRevisionsSuccessMsg:
		p.sync()
	}
	return p.m.Update(msg)
}

func (p *bookmarksPanel) sync() {
	if !p.s.hasCore {
		return
	}
	current := ""
	if rev := p.s.core.Revisions.SelectedRevision(); rev != nil {
		current = rev.CommitId
	}
	p.m.SyncRevisionContext(current, p.s.core.Revisions.GetCommitIds())
}

func (p *bookmarksPanel) draw(dl *render.DisplayContext, inner layout.Box) { p.m.ViewRect(dl, inner) }
func (p *bookmarksPanel) setFocused(focused bool)                          { p.m.SetFocused(focused) }
func (p *bookmarksPanel) editing() bool                                    { return p.m.IsEditing() }

func (p *bookmarksPanel) scopes() []common.Scope {
	if p.m.IsEditing() {
		return p.m.Scopes()
	}
	return append([]common.Scope{{Name: actions.ScopeJjjBookmarks, Leak: common.LeakAll, Handler: p.s.h}}, p.m.Scopes()...)
}

func (p *bookmarksPanel) handleIntent(intent intents.Intent) (tea.Cmd, bool) {
	if it, ok := intent.(intents.BookmarkPaneAdvance); ok {
		return p.m.Advance(it.ToWorkingCopy), true
	}
	return nil, false
}
```

In `New` (shell.go), after registering Files:

```go
	s.register(PanelBookmarks, &bookmarksPanel{s: s, m: bookmarkpane.New(ctx)})
```

In `Update`, add to the type switch (import `"github.com/idursun/jjui/internal/ui/bookmarkpane"`):

```go
	case bookmarkpane.PaneClickedMsg, bookmarkpane.ItemClickedMsg, bookmarkpane.RemoteClickedMsg:
		s.setFocus(PanelBookmarks)
		return s.panels[PanelBookmarks].update(msg), true
	case bookmarkpane.RevealRevisionMsg, bookmarkpane.BeginMoveBookmarkMsg, bookmarkpane.BeginCreateBookmarkMsg:
		// upstream (ui.handleSplitMsg) reveals the revision or starts the
		// move/create operation on the revisions list; follow it there.
		s.setFocus(PanelJournal)
		return nil, false
```

`setFocus` already calls `p.setFocused(pid == id)` for every panel, which keeps the pane's own `Focused()` (and so its `Scopes()`) in step with the shell.

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/ui/shell ./internal/ui/bookmarkpane -v`
Expected: PASS.

- [ ] **Step 8: Record and commit**

Add to `UPSTREAM.md`'s table:

```markdown
| `internal/ui/bookmarkpane/advance.go` | new file: `Model.Advance` (`jj bookmark advance`, optional push) for the jjj Bookmarks panel; uses the pane's private selection helpers; upstream PR candidate |
```

Run: `make check`
Expected: all green.

```bash
jj commit -m "feat(jjj): Bookmarks panel hosted persistently, a/A advance (TASK-012.01)"
```

---

### Task 8: Docs, smoke, follow-up tasks

**Files:**
- Modify: `docs/jjj.md` (key reference), `scripts/pty-smoke.py` is used as-is
- Backlog: two new tasks; TASK-006 and TASK-012.01 closed

- [ ] **Step 1: Update the quickstart**

In `docs/jjj.md`, replace the key table (or add one if absent) with the defaults from Task 2's `bindings`, grouped as Shell / Detail / Notes / Files / Bookmarks, and add this paragraph:

```markdown
Every jjj key is an action: rebind it in `~/.config/jjj/config.toml` like any
jjui binding, e.g. `bindings = [{ key = "x", action = "jjj.shell.focus_files", scope = "jjj.shell" }]`.
Files keeps upstream's file-details keys (space select, r restore, s split,
S squash, A absorb, d diff, / filter, * revisions touching the file);
Bookmarks keeps upstream's bookmark-pane keys (c create, m move, d delete,
r rename, t/u track, P push, F fetch, e edit, n new, s show).
```

In `UPSTREAM.md` → "Key collisions and forced settings", replace the first paragraph (raw interception) with:

```markdown
jjj keys are actions in jjj scopes (`internal/jjj/defaults.toml`). The focused
panel's scope comes first, then `jjj.shell` (LeakAll, Global), then upstream's
scopes. Collisions resolved by order: `tab`/`shift+tab` (upstream `ui.focus_next_pane`,
and `revisions.details` `tab`) and digits are claimed by jjj while no upstream
operation is open; inside Files, `tab` is bound in `jjj.files` because the
details scope sits between it and `jjj.shell`.
```

- [ ] **Step 2: Manual PTY smoke with the new panels**

Run:

```bash
make build
python3 scripts/pty-smoke.py bin/jjj $HOME/Developer/Home/jjjournal --wait 12 \
  --expect "[2]─Files" --expect "[3]─Bookmarks" --expect "[6]─Notes" --expect "revset:"
```

Expected: exit 0.

Then run `bin/jjj` in `~/Developer/Home/jjjournal` by hand at ≥ 120 columns and check, noting any failure as a backlog bug rather than skipping it:
- `2` focuses Files; `j`/`k` move the file cursor and Detail shows that file's diff; `esc` returns to Journal.
- With Files focused, `a` does nothing to the selected revision (no abandon dialog).
- `3` focuses Bookmarks; `a` advances the closest bookmark to the selected revision and the flash shows the jj output; `u` (back in Journal) undoes it.
- `6` focuses Notes; `enter` opens inline describe on the Journal row.
- `+` twice zooms the focused panel to full screen; `_` twice restores.
- At < 120 columns, `2` shows Files full-screen and `esc` returns to the list.

- [ ] **Step 3: File follow-ups and close tasks**

```bash
backlog task create "Files panel: tree view toggle (t)" -p TASK-012 --desc "Deferred from P2a (plan deviation 3): needs a tree mode in upstream details_list.go or a jjj-side renderer over DetailsList." -l feature --priority low --plain
backlog task create "Bookmarks panel: @ marker for bookmarks at the selected revision" -p TASK-012 --desc "Deferred from P2a (plan deviation 3): needs an edit to upstream bookmarkpane/view.go; the pane already sorts those bookmarks first." -l feature --priority low --plain
backlog task edit TASK-006 -s Done --append-notes "Done in TASK-012.01: shell keys are actions in jjj.shell (internal/jjj/defaults.toml)." --plain
backlog task edit TASK-012.01 -s Done --append-notes "P2a shipped per docs/superpowers/plans/2026-10-13-jjj-p2a-panels.md." --plain
```

- [ ] **Step 4: Final check and commit**

Run: `make check`
Expected: all green.

```bash
jj commit -m "docs(jjj): P2a key reference, upstream record; close TASK-006, TASK-012.01"
```
