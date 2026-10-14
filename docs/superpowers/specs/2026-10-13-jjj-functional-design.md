# jjj functional layer — Design

- **Date:** 2026-10-13
- **Status:** approved in conversation, awaiting written-spec review
- **Builds on:** `2026-09-04-jjjournal-design.md` (the base spec). Section
  numbers prefixed "base §" refer to it.
- **Backlog:** TASK-012 (umbrella). Supersedes the scope of TASK-001 (P2 panels)
  and closes TASK-006 (rebindable shell keys).

## 1. Purpose

jjj today shows three panels — Status, Journal, Detail. Every upstream jjui
operation works inside Journal, but nothing on screen says so, files are only
visible in a modal, bookmarks only in a toggled side split, and a good part of
jj's command set (arrange, bisect, converge, fix, interdiff, next/prev,
bookmark advance, …) is unreachable except by typing it after `:`.

This spec makes jjj feel like a jj tool as well as a journal, in three parts:

1. **Panels** — live Files and Bookmarks as main panels, Notes, accordion and
   zoom (the base spec's P2, widened).
2. **Command layer** — every jj command reachable and discoverable through a
   context-aware palette, a hint bar, and a few native multi-step flows.
3. **Review mode** — walk a stack before pushing, or a day/week after the fact,
   file by file, with marks and warnings.

Order matters: review is built on the Files panel and the command layer.

## 2. Decisions

| Topic | Decision |
|---|---|
| Panels | Files (2), Bookmarks (3) and Notes (6) are permanent main panels, not modals or toggles. |
| "Review" means | (a) stack review before push and (b) period review (day / week) after the fact. Reviewing other people's incoming changes is out of scope. |
| Command surface | Approach A: context-aware palette + hint bar + native flows for multi-step commands. Not one key per command; not Lua-only. |
| Interactive jj commands | `arrange`, `converge`, `diffedit`, `resolve`, `split -i` run attached to the terminal (jjj suspends); jjj does not reimplement their UIs. |
| Spec shape | One spec, three phases (P2a/P2b/P2c), one implementation plan per phase. |

## 3. Foundation: scoped key routing for shell panels

### 3.1 The seam

Today the shell intercepts raw keys (`shell.HandleKey`) while upstream is in
normal mode. That works for shell-level keys but not for panels with their own
keymaps, and the project rule is that models handle `intents.Intent` through
scopes.

Add to `ui.Extension`:

```go
// Scopes returns the extension's dispatch scopes, innermost first. They are
// placed before the revisions scopes in dispatchScopes; an empty slice means
// the extension claims no keys this frame.
Scopes() []common.Scope
```

`ui.Model.dispatchScopes()` inserts `m.ext.Scopes()` immediately before the
primary (`revisions` / `oplog` / split) scopes and after modal scopes
(password, revset editing, diff, stacked). Modals therefore still win.

`inNormalMode()` gains a clause so raw `HandleKey` is not offered while a shell
panel is in an editing state (a filter input, the palette input, a confirmation).
The shell exposes that through a method on the extension (`Editing() bool`), so
`inNormalMode` stays one line longer, not restructured.

### 3.2 jjj intents

`cmd/genactions` scans only `internal/ui/intents/`. jjj's intent types and
`//jjui:bind` annotations go in **new files** there, all prefixed `jjj_`
(`jjj_shell_intents.go`, `jjj_files_intents.go`, `jjj_palette_intents.go`,
`jjj_review_intents.go`). No existing upstream intent file is edited.

Consequences:

- jjj actions are bindable in `config.toml` like any jjui action. The shell-level
  keys (`0`–`9`, `tab`, `shift+tab`, `J`, `{`, `}`, `+`, `_`) become actions in a
  `shell` scope, closing TASK-006. The raw-key interception in `shell.HandleKey`
  is removed except for what cannot be scoped (none expected; if one remains it
  is listed in `UPSTREAM.md`).
- The generated files (`internal/ui/actions/catalog_gen.go`,
  `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua`)
  contain jjj entries. After an upstream sync they are **regenerated, never
  merged by hand** (`go run ./cmd/genactions`); this step is added to the sync
  procedure in `UPSTREAM.md`.
- Default bindings for jjj actions live in jjj's existing compiled-in defaults
  (`internal/jjj/defaults.toml`, already layered over upstream's defaults and
  under the user's config by `jjj.ApplyDefaults`), not in upstream's
  `internal/config/default/bindings.toml`.

### 3.3 Focus and leak policy

- The focused panel's scope is innermost.
- **Journal focused:** no shell panel scope; upstream's revisions scopes handle
  keys exactly as today.
- **Files / Bookmarks / Notes / Detail focused:** that panel's scope, then the
  `shell` scope, then upstream's global `ui` scope (quit, help, undo, redo,
  oplog). Leak policy `LeakGlobal`: the **revisions** scope is *not* visible, so
  `a` in Files can never abandon a commit.
- Every panel reads the **current revision** from Journal's selection
  (`ctx.SelectedItem`), so revision-scoped commands act on the right change from
  anywhere.

## 4. Panels (P2a)

### 4.1 Layout

```
┌[1] Status─────────┐┌[0] Detail────────────────────────────┐
├[2] Files──────────┤│ show --stat / one file's diff         │
│ M ui/shell.go +12 ││ / annotate / review content           │
├[3] Bookmarks──────┤│                                       │
│ main ↑2 origin    ││                                       │
├[4] Journal────────┤├[6] Notes──────────────────────────────┤
│ …tallest…         ││ full description, wrapped             │
└───────────────────┘└───────────────────────────────────────┘
 hint bar: keys for the focused panel  ·  x commands
```

- Left column: Status (fixed 5 rows), Files, Bookmarks, Journal. Tasks (5)
  joins in P3, Stats (7) in P4, per the base spec.
- **Accordion:** the focused left panel takes ~50 % of the column's flexible
  height; the others share the rest, minimum 3 rows each. Journal never drops
  below 40 % while unfocused.
- Right column: Detail takes what Notes does not; Notes is `[shell] notes_rows`
  (default 8) rows.
- **Zoom:** `+` cycles the focused panel normal → full column → full screen;
  `_` reverses. Zoom state resets when focus moves.
- **Narrow fallback** (below `[shell] narrow_columns`, default 120): the stock
  single-list layout (as today); a digit opens that panel zoomed full-screen,
  `esc` returns to the list.
- `[shell] hidden = ["files", …]` removes panels from the layout and the tab cycle.

### 4.2 Files (panel 2)

Files changed in the Journal-selected revision; for `@`, the working-copy
changes. Reuses upstream's `operations/details.DetailsList` (list, parsing,
selection, filter) hosted as a persistent model in a new package
`internal/ui/files`, rather than the modal `details.Operation`.

- **Row:** status letter (`A` `M` `D` `R` `C`), path, `+n −m`, conflict marker.
- **Views:** flat (default) and tree (`t` toggles; directories collapsible with
  `enter`/`h`/`l`).
- **Detail coupling:** while Files is focused, Detail shows the cursor file's
  diff (`[preview] file_command`); leaving Files restores the revision view.
- **Reload:** on `common.RefreshMsg` and on selection change, debounced as
  Status already does (base spec, P1 final review).

| Key | Action | Source |
|---|---|---|
| `j`/`k`, `pgup`/`pgdown` | move | upstream details |
| `space` | toggle select | upstream details |
| `/` | filter | upstream details |
| `r` | restore selected files (confirm) | upstream details |
| `s` / `alt+s` | split on selected files / parallel | upstream details |
| `S` | squash selected files into parent | upstream details |
| `A` | absorb selected files | upstream details |
| `d` | full-screen diff | upstream details |
| `*` | revisions changing this file (sets revset) | upstream details |
| `enter` | focus Detail on this file's diff | new |
| `e` | `jj diffedit -r <rev> -- <files>` (terminal) | new |
| `o` | open file in `$EDITOR` (working-copy path; `@` only, otherwise flash) | new |
| `b` | `jj file annotate -r <rev> <file>` → Detail | new |
| `m` | `jj resolve -r <rev> <file>` (terminal; conflicted files only) | new |
| `t` | flat ↔ tree | new |

### 4.3 Bookmarks (panel 3)

Upstream's `bookmarkpane.Model` hosted permanently in the shell instead of the
split container. Its scope and bindings are unchanged (`c` create, `m` move,
`d` delete, `f` forget, `r` rename, `t`/`u` track/untrack, `P` push, `F` fetch,
`e` edit, `n` new, `s` show in Journal, `alt+enter` set revset, `space` select,
`/` filter, `h`/`l` cycle remotes). The shell forwards
`bookmarkpane.RevisionContextMsg` on selection change, as `ui.syncBookmarkPaneContext`
does for the split, and handles the pane's outbound messages
(`RevealRevisionMsg`, `BeginMoveBookmarkMsg`, `BeginCreateBookmarkMsg`) by
moving focus to Journal first, mirroring `ui.handleSplitMsg`.

New:

- **`a` advance** — `jj bookmark advance --to <rev>`, where `<rev>` is the
  Journal-selected revision. With bookmarks multi-selected, they are passed by
  name; with none selected, jj's own default (closest bookmarks) applies. Flash
  shows the moved bookmarks and "`u` to undo".
- **`A` advance to `@`, then offer push** — advance with `--to @`, then a
  confirmation "push <names> to <remote>?" that runs upstream's push.
- **`@` marker** on bookmarks whose target is the Journal-selected revision.

Advance is implemented as a new intent (`BookmarkPaneAdvance`, with a `ToWorkingCopy`
field) in `jjj_bookmarks_intents.go` and a handler added to `bookmarkpane`. This is
the one edit to an upstream model in this spec; it is written to be sent upstream
as a PR and is listed in `UPSTREAM.md`.

### 4.4 Notes (panel 6)

New package `internal/ui/notes`. The selected revision's full description,
soft-wrapped, first line bold; `(no description)` when empty. In review mode it
also lists the revision's warnings (§6.4). `enter` opens upstream's inline
describe for that revision (focus moves to Journal while it is open). `j`/`k`
scroll.

### 4.5 Status and Detail

Unchanged, plus: Status shows `⚠ divergent n` when `divergent()` is non-empty
(one extra `jj log -r 'divergent()' -T '"x"' --no-graph` count in the existing
debounced reload); Detail can show annotate output (§4.2) and review content (§6).

## 5. Command layer (P2b)

### 5.1 Palette

`x` (also `ctrl+p`) opens a fuzzy palette, a new model in `internal/ui/palette`
rendered as an overlay. `:` stays upstream's free-form `exec jj`.

- Lists the command-table entries (§5.2) whose context matches the focused panel
  and selection, plus the user's upstream `[[actions]]` (label = action name).
- Row: name, one-line description, the direct key if bound (looked up from the
  live bindings, so rebinding shows correctly).
- `enter` runs; `esc` closes. Typing filters (upstream `fuzzy_search` matcher).

### 5.2 Command table

New package `internal/jjcmd`: pure data plus argument expansion, no UI.

```go
type Context uint8 // bit set: Revision | File | Bookmark | Global
type Kind uint8    // Delegate | Run | Terminal | Flow

type Command struct {
    Name     string     // "next --edit"
    Desc     string
    Contexts Context
    Kind     Kind
    Args     []string   // template: $change_id $commit_id $file $files $bookmark $selected
    Intent   intents.Intent // Kind == Delegate: the upstream intent to dispatch
    Flow     string     // Kind == Flow: flow id
    Confirm  string     // non-empty → confirmation text before running
    Needs    func(Selection) bool // optional extra guard (e.g. conflicted file)
}
```

`Selection` carries the current revision, multi-selected revisions, the Files
cursor and selection, and the Bookmarks cursor and selection. Expansion fails
(entry hidden) when a template variable has no value.

| Kind | Behaviour | Commands |
|---|---|---|
| **Delegate** | dispatch the existing upstream intent (opens the upstream operation) | rebase, squash, split, abandon, absorb, duplicate, revert, new, commit, describe, edit, evolog, set-parents, undo, redo, op log, git push, git fetch, diff range |
| **Run** | `jj …` in the background via the command runner; flash result; refresh | next, prev, next --edit, prev --edit, fix -s, simplify-parents, parallelize (selection), metaedit --update-author, metaedit --reset-author, metaedit --update-change-id, sign, unsign, tag set, tag delete, bookmark advance, restore --from |
| **Terminal** | suspend jjj (`tea.ExecProcess`), run attached, resume, refresh | arrange, converge, diffedit, resolve, split -i |
| **Flow** | jjj-native multi-step interaction (§5.3) | interdiff, bisect, run |

Destructive entries (abandon, restore, converge, metaedit --update-change-id,
tag delete) carry `Confirm` and go through upstream's `confirmation` model.
Every Run/Terminal success flash ends with "`u` to undo".

### 5.3 Flows

All "pick another revision" steps reuse upstream's `operations/target_picker`.

- **interdiff:** A = selected revision, pick B → Detail shows
  `jj interdiff --color always --from A --to B --git`.
- **bisect:** good = selected revision, pick bad (default `@`) → input "test
  command (empty = interactive shell)" → Terminal:
  `jj bisect run --range good..bad -- <cmd>` (empty → `$SHELL`). On return,
  parse the first-bad change id from jj's output and reveal it in Journal
  (`revisions.RevealRevision`); if parsing fails, flash the last output line.
- **run:** input command → Background: `jj run -r <selection or revset> -- <cmd>`;
  output summary to Detail.
- **converge** is a Terminal entry (jj prompts on its own), offered for
  revisions in `divergent()`; the Status marker (§4.5) is the prompt to use it.

### 5.4 Hint bar

When the shell is active, the bottom line shows the 6–8 highest-priority
bindings for the focused panel's scope (priority = an ordered list per scope in
`internal/jjj/defaults.toml` under `[shell.hints]`, falling back to declaration order), then
`x more…`. It replaces upstream's long status-line help in shell mode; flash
messages still render on the same line and take precedence while shown.
`[shell] hint_bar = false` restores upstream's status line.

## 6. Review mode (P2c)

### 6.1 Entering and leaving

Review is a mode of the Journal panel, not a new panel. `V` (free upstream)
opens a one-line chooser: `s` stack · `d` day · `w` week. `V` on a day or week
header goes straight to that period. While active:

- The Journal title reads `[4] Review · <scope> <done>/<total> ✓`.
- The row source shows only the review revset, **oldest first** (topological for
  stack, committer time for period).
- Files, Detail, Notes follow the cursor as in normal mode.
- `esc` or `V` leaves and restores the previous revset and journal/graph mode.

The previous revset/mode are saved by the shell, not by upstream; the review
revset is applied through the same row-source factory as journal mode.

### 6.2 Scopes

| Scope | Revset | Config |
|---|---|---|
| Stack | `trunk()..@ ~ (empty() & description(exact:""))` | `[review] stack_revset` |
| Day | the calendar day under the cursor (journal day boundaries, local time) | — |
| Week | the ISO week under the cursor (Monday start) | — |

Day/week revsets are built in Go from `internal/journal`'s boundary helpers as
`committer_date(after:"…", before:"…")`.

### 6.3 Keys

| Key | Action |
|---|---|
| `n` / `p` | next / previous commit |
| `]` / `[` | next / previous file within the commit; at the last file, moves to the first file of the next commit. Detail shows that file's diff. |
| `space` | toggle reviewed (✓) on the commit and advance to the next unreviewed |
| `P` | stack finish (§6.6) |
| `E` | period finish (§6.6) |
| everything else | the normal Journal keys and the `x` palette (describe, squash, absorb, split, …) |

### 6.4 Warnings

Computed per commit in `internal/review`; shown as a `!` marker on the row and
listed in Notes.

- no description
- empty change
- conflict
- divergent
- description matches `[review] warn_pattern` (default
  `(?i)\b(wip|todo|fixme)\b|^fixup!`)
- a changed file contains conflict markers (`<<<<<<<`, `>>>>>>>` at line start)
  in the diff's added lines
- stack scope only: not reachable from any local bookmark after the stack top
  (i.e. would not be pushed by advancing the closest bookmark) — informational

Blocking warnings (for `P`): conflict, no description, conflict markers.

### 6.5 Marks

- Keyed by **commit id**. Rewriting a commit gives it a new commit id, so its
  mark disappears — rewritten code needs re-review.
- Stored in `.jj/jjj/review.json` (`{"<commit_id>": "<RFC3339 time>"}`), inside
  `.jj/` so never tracked. Written on every toggle; marks for commits no longer
  visible are pruned on load.
- Missing or corrupt file → start empty, flash once.

### 6.6 Summary and finish

A one-line summary heads the Journal panel in review mode:

- Stack: `6 commits · 14 files · +412 −88 · 2 warnings · main ↑6`
- Period: `Tue 2026-10-13 · 9 changes · 3 heads · 21 files · +600 −140 · bookmarks moved: main`

Totals come from one `jj log --no-graph -r <revset> -T <stat template>` per
reload.

- **`P` (stack):** if every commit is ✓ and no blocking warnings: confirmation
  listing the exact commands —
  `jj bookmark advance --to <stack top>` then `jj git push -b <names>` — and run
  them on confirm. Otherwise flash what blocks (counts of unreviewed and of each
  blocking warning).
- **`E` (period):** exports the period to the daily note via P5's exporter
  (base §10). Until P5 ships, `E` flashes "export arrives with P5".

### 6.7 Packages

- `internal/review` — pure: revset building, warning checks, progress, mark
  store (load/save/prune). No UI imports.
- `internal/ui/review` — row decorations, summary header, review scope and
  intents, mode switching via the shell.

## 7. Configuration additions

```toml
[shell]
hint_bar = true          # false → upstream status-line help

[review]
stack_revset = "trunk()..@ ~ (empty() & description(exact:\"\"))"
warn_pattern = "(?i)\\b(wip|todo|fixme)\\b|^fixup!"
```

The palette key and every jjj key are ordinary actions, rebound through the
normal `[[bindings]]` config.

## 8. Upstream bookkeeping

Added to `UPSTREAM.md`'s touched-files table:

| File | Reason |
|---|---|
| `internal/ui/ui.go` | `Extension.Scopes()` and `Extension.Editing()`; one insertion in `dispatchScopes`, one clause in `inNormalMode` |
| `internal/ui/intents/jjj_*.go` | new files (jjj intents + annotations); no upstream intent file edited |
| `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua` | regenerated with jjj entries; regenerate after every sync, never hand-merge |
| `internal/ui/bookmarkpane/*` | advance intent handler (upstream PR candidate) |

Sync procedure gains: `go run ./cmd/genactions` after the rebase, before
`make check`.

## 9. Testing

- **Unit:**
  - `internal/jjcmd`: template expansion (each variable, missing-variable
    hiding), context filtering, multi-select argument building.
  - `internal/review`: stack and period revsets (DST, year boundary), each
    warning, blocking set, progress counts, mark invalidation on commit-id
    change, store round-trip and prune, corrupt-file recovery.
  - `internal/ui/shell`: layout with five left panels, accordion, Journal 40 %
    floor, zoom cycle, hidden panels, narrow fallback.
  - Scope ordering: with Files focused, `a` resolves to no revisions intent; with
    Journal focused, upstream intents resolve unchanged.
- **Integration** (style of `internal/ui/bookmarkpane_integration_test.go`, with
  the test command runner): advance runs the expected `jj bookmark advance`
  args; Files restore; palette Run entry executes; Delegate entry opens the
  upstream operation; review `space` writes a mark.
- **End-to-end:** one PTY scenario per phase in `e2e/` (focus each panel by
  number, advance a bookmark, open the palette and run `next`, run a stack
  review to `P` against a local bare remote).
- **Upstream intact:** `make check` green on every commit.

## 10. Delivery

| Phase | Deliverable | Done when |
|---|---|---|
| **P2a: seam + panels** | §3 seam and jjj intents (closes TASK-006); Files, Bookmarks with advance, Notes; accordion, zoom, narrow fallback | all left panels live; `a` advances; Files follows the Journal cursor; `make check` green |
| **P2b: command layer** | `internal/jjcmd`, palette, hint bar, four run kinds, interdiff/bisect/run flows, divergence marker | every command in §5.2 is reachable from the palette; bound ones show their key |
| **P2c: review** | `V` stack + period review, marks, warnings, summary, `P` finish | a stack review run end-to-end in a scratch repo, finishing with advance + push to a local bare remote |

P3 (Tasks), P4 (Stats), P5 (export), P6 (sync) keep their base-spec places.
Review's `E` activates with P5.

## 11. Out of scope

- Reviewing other people's incoming changes (post-fetch walk).
- Reimplementing jj's interactive UIs (arrange, converge prompts, diff editors).
- Hunk-level staging inside Files (jj's model is split/squash/diffedit; those are
  covered).
- Gerrit, sparse, workspace management beyond what `:` already allows.
- Any change to stock jjui's behaviour; everything here is additive or behind
  the Extension.
